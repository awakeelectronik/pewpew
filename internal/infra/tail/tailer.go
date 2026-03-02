package tail

import (
	"bufio"
	"context"
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
}

// NewSSHTailer crea un nuevo tailer eficiente
func NewSSHTailer(
	logPath string,
	db store.Store,
	geoip geoip.Resolver,
	parser func(line string) *parse.SSHEvent,
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

// readLoop lee líneas en streaming (eficiente en RAM)
func (t *SSHTailer) readLoop(ctx context.Context) {
	defer t.wg.Done()

	reader := bufio.NewReader(t.file)
	checkTicker := time.NewTicker(1 * time.Second)
	defer checkTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		case <-checkTicker.C:
			// Intentar leer línea
			line, err := reader.ReadString('\n')
			if err != nil {
				// EOF o error; esperar y reintentar (por rotación de logs)
				continue
			}

			if len(line) == 0 {
				continue
			}

			// Parse SSH event
			sshEvent := t.parser(line)
			if sshEvent == nil {
				continue
			}

			// Enrich con GeoIP
			geoData := t.geoip.Resolve(sshEvent.IP)

			// Crear event
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

			// Enviar a batch processor
			select {
			case t.eventsCh <- event:
			case <-t.stopCh:
				return
			}
		}
	}
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
