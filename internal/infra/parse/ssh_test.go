package parse

import "testing"

func TestParseSSHLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"failed", "sshd[1]: Failed password for root from 1.2.3.4 port 22 ssh2", "failed_password"},
		{"invalid", "sshd[1]: Invalid user admin from 4.3.2.1 port 22", "invalid_user"},
		{"accepted", "sshd[1]: Accepted publickey for root from 8.8.8.8 port 22 ssh2", "accepted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSSHLine(tt.line)
			if got == nil || got.EventType != tt.want {
				t.Fatalf("expected %s, got %#v", tt.want, got)
			}
		})
	}
}

func BenchmarkParseSSHLine(b *testing.B) {
	line := "sshd[1]: Failed password for root from 210.79.190.209 port 58356 ssh2"
	for i := 0; i < b.N; i++ {
		_ = ParseSSHLine(line)
	}
}
