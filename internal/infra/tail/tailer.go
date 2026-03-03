package tail

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"pewpew/internal/domain"
	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/parse"
	"pewpew/internal/infra/store"
)

// SSHTailer lee /var/log/auth.log en streaming sin cargar todo en RAM
type SSHTailer struct {
	logPath    string
	file       *os.File
	db         store.Store
	geoip      geoip.Resolver
	parser     func(line string) *parse.SSHEvent
	eventsCh   chan *domain.SecurityEvent
	batchSize  int
	batchTime  time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
	onEvent    func(event *domain.SecurityEvent)
}

// NewSSHTailer crea un nuevo tailer eficiente
func NewSSHTailer(
	logPath string,
	db store.Store,
	geoip geoip.Resolver,
	parser func(line string) *parse.SSHEvent,
	onEvent func(event *domain.SecurityEvent),
) *SSHTailer {
	return &SSHTailer{
		logPath:   logPath,
		db:        db,
		geoip:     geoip,
		parser:    parser,
		eventsCh:  make(chan *domain.SecurityEvent, 256), // Buffer limitado
		batchSize: 100,                                     // Batch cada 100 eventos
		batchTime: 500 * time.Millisecond,                 // O cada 500ms
		stopCh:    make(chan struct{}),
		onEvent:   onEvent,
	}
}

// Start inicia el tailer
func (t *SSHTailer) Start(ctx context.Context) error {
	// Abrir archivo
	file, err := os.Open(t.logPath)
	if err != nil {
		return err
	}
	t.file = file

	// Ir al final del archivo (solo nuevas líneas)
	if _, err := file.Seek(0, 2); err != nil {
		return err
	}

	// Lector + parser
	t.wg.Add(1)
	go t.readLoop(ctx)

	// Batch processor
	t.wg.Add(1)
	go t.batchProcessor(ctx)

	log.Printf("tailer started on %s", t.logPath)
	return nil
}

// Stop detiene el tailer
func (t *SSHTailer) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	if t.file != nil {
		t.file.Close()
	}
}

// readLoop lee líneas en streaming (eficiente en RAM). Detecta rotación del archivo
// (p. ej. logrotate) y reabre el archivo para seguir leyendo sin perder líneas.
func (t *SSHTailer) readLoop(ctx context.Context) {
	defer t.wg.Done()

	const rotationCheckInterval = 2 * time.Second
	rotationTicker := time.NewTicker(rotationCheckInterval)
	defer rotationTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		default:
		}

		// Reabrir si el archivo fue rotado (inode distinto)
		if t.shouldReopen() {
			t.reopenFile()
		}

		reader := bufio.NewReader(t.file)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Sin más datos; esperar un poco y revisar rotación
				select {
				case <-ctx.Done():
					return
				case <-t.stopCh:
					return
				case <-rotationTicker.C:
				}
				continue
			}
			// Error de lectura (archivo cerrado/rotado): reabrir en la siguiente iteración
			continue
		}

		if len(line) == 0 {
			continue
		}

		sshEvent := t.parser(line)
		if sshEvent == nil {
			continue
		}

		geoData := t.geoip.Resolve(sshEvent.IP)
		event := &domain.SecurityEvent{
			Timestamp: time.Now(),
			Source:    "ssh",
			EventType: sshEvent.EventType,
			IP:        sshEvent.IP,
			Username:  sshEvent.Username,
			Port:      sshEvent.Port,
			Message:   line,
			Country:   geoData.CountryCode,
			City:      geoData.City,
			Latitude:  geoData.Latitude,
			Longitude: geoData.Longitude,
		}

		select {
		case t.eventsCh <- event:
			if t.onEvent != nil {
				t.onEvent(event)
			}
		case <-t.stopCh:
			return
		}
	}
}

// shouldReopen devuelve true si el archivo en disco (por path) es distinto al que tenemos abierto (rotación).
func (t *SSHTailer) shouldReopen() bool {
	if t.file == nil {
		return true
	}
	ourStat, err := t.file.Stat()
	if err != nil {
		return true
	}
	pathStat, err := os.Stat(t.logPath)
	if err != nil {
		return false
	}
	return !os.SameFile(ourStat, pathStat)
}

// reopenFile cierra el archivo actual y abre de nuevo por path (seek al final para solo nuevas líneas).
func (t *SSHTailer) reopenFile() {
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}
	file, err := os.Open(t.logPath)
	if err != nil {
		log.Printf("tailer reopen failed: %v", err)
		return
	}
	if _, err := file.Seek(0, 2); err != nil {
		file.Close()
		return
	}
	t.file = file
	log.Printf("tailer reopened %s (log rotation)", t.logPath)
}

// batchProcessor agrupa eventos y los guarda en lotes (reduce I/O)
func (t *SSHTailer) batchProcessor(ctx context.Context) {
	defer t.wg.Done()

	var batch []*domain.SecurityEvent
	batchTicker := time.NewTicker(t.batchTime)
	defer batchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.flushBatch(batch)
			return
		case <-t.stopCh:
			t.flushBatch(batch)
			return
		case event := <-t.eventsCh:
			batch = append(batch, event)
			if len(batch) >= t.batchSize {
				t.flushBatch(batch)
				batch = batch[:0]
				batchTicker.Reset(t.batchTime)
			}
		case <-batchTicker.C:
			if len(batch) > 0 {
				t.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// flushBatch guarda un lote en DB (transacción)
func (t *SSHTailer) flushBatch(events []*domain.SecurityEvent) {
	if len(events) == 0 {
		return
	}

	if err := t.db.InsertEventsBatch(events); err != nil {
		log.Printf("error inserting batch: %v", err)
	}
}
