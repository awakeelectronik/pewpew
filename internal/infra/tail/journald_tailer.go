package tail

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"sync"
	"time"

	"pewpew/internal/domain"
	"pewpew/internal/infra/geoip"
	"pewpew/internal/infra/parse"
	"pewpew/internal/infra/store"
)

// SSHJournalTailer consume eventos SSH desde journald.
type SSHJournalTailer struct {
	db        store.Store
	geoip     geoip.Resolver
	parser    func(line string) *parse.SSHEvent
	eventsCh  chan *domain.SecurityEvent
	batchSize int
	batchTime time.Duration
	stopCh    chan struct{}
	wg        sync.WaitGroup
	onEvent   func(event *domain.SecurityEvent)
}

func NewSSHJournalTailer(
	db store.Store,
	geoip geoip.Resolver,
	parser func(line string) *parse.SSHEvent,
	onEvent func(event *domain.SecurityEvent),
) *SSHJournalTailer {
	return &SSHJournalTailer{
		db:        db,
		geoip:     geoip,
		parser:    parser,
		eventsCh:  make(chan *domain.SecurityEvent, 256),
		batchSize: 100,
		batchTime: 500 * time.Millisecond,
		stopCh:    make(chan struct{}),
		onEvent:   onEvent,
	}
}

func (t *SSHJournalTailer) Start(ctx context.Context) error {
	t.wg.Add(1)
	go t.readLoop(ctx)
	t.wg.Add(1)
	go t.batchProcessor(ctx)
	log.Printf("journald tailer started for ssh.service")
	return nil
}

func (t *SSHJournalTailer) Stop() {
	close(t.stopCh)
	t.wg.Wait()
}

func (t *SSHJournalTailer) readLoop(ctx context.Context) {
	defer t.wg.Done()

	cmd := exec.CommandContext(ctx, "journalctl", "-u", "ssh", "-f", "-n", "0", "--output=short-iso")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("journald stdout pipe error: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("journald start error: %v", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return
		case <-t.stopCh:
			_ = cmd.Process.Kill()
			return
		default:
		}

		line := scanner.Text()
		sshEvent := t.parser(line)
		if sshEvent == nil {
			continue
		}

		geoData := t.geoip.Resolve(sshEvent.IP)
		event := &domain.SecurityEvent{
			Timestamp: time.Now(),
			Source:    "ssh_journald",
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
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return
		case <-t.stopCh:
			_ = cmd.Process.Kill()
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("journald scanner error: %v", err)
	}
}

func (t *SSHJournalTailer) batchProcessor(ctx context.Context) {
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

func (t *SSHJournalTailer) flushBatch(events []*domain.SecurityEvent) {
	if len(events) == 0 {
		return
	}
	if err := t.db.InsertEventsBatch(events); err != nil {
		log.Printf("journald batch insert error: %v", err)
	}
}
