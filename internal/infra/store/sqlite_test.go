package store

import (
	"path/filepath"
	"testing"
	"time"

	"pewpew/internal/domain"
)

func TestSQLiteStoreEventsAndAttackers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}

	err = s.InsertEventsBatch([]*domain.SecurityEvent{
		{Timestamp: time.Now(), Source: "ssh", EventType: "failed_password", IP: "1.1.1.1", Message: "x"},
		{Timestamp: time.Now(), Source: "ssh", EventType: "failed_password", IP: "1.1.1.1", Message: "y"},
		{Timestamp: time.Now(), Source: "ssh", EventType: "invalid_user", IP: "2.2.2.2", Message: "z"},
	})
	if err != nil {
		t.Fatal(err)
	}

	attackers, err := s.GetTopAttackers(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(attackers) == 0 || attackers[0].IP != "1.1.1.1" {
		t.Fatalf("unexpected attackers: %#v", attackers)
	}
}

func TestTopAttackersLastSeen(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}

	seen := time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC)
	err = s.InsertEventsBatch([]*domain.SecurityEvent{
		{Timestamp: seen.Add(-time.Hour), Source: "ssh", EventType: "failed_password", IP: "1.1.1.1", Message: "x"},
		{Timestamp: seen, Source: "ssh", EventType: "failed_password", IP: "1.1.1.1", Message: "y"},
	})
	if err != nil {
		t.Fatal(err)
	}

	attackers, err := s.GetTopAttackers(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(attackers) != 1 || !attackers[0].LastSeen.Equal(seen) {
		t.Fatalf("unexpected last seen: %#v", attackers)
	}
}
