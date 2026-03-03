package parse

import (
	"net"
	"regexp"
	"strconv"
	"strings"
)

// SSHEvent es el resultado del parse
type SSHEvent struct {
	EventType string
	IP        string
	Username  string
	Port      int
}

// Regex patterns para SSH
var (
	failedPasswordRe = regexp.MustCompile(`Failed password for (?:invalid user )?(\S+) from ([0-9a-f:.]+) port (\d+)`)
	invalidUserRe    = regexp.MustCompile(`Invalid user (\S+) from ([0-9a-f:.]+)`)
	acceptedRe       = regexp.MustCompile(`Accepted (?:password|publickey) for (\S+) from ([0-9a-f:.]+) port (\d+)`)
	kexRe            = regexp.MustCompile(`kex_exchange_identification.*?([0-9a-f:.]+)?`)
	disconnectRe     = regexp.MustCompile(`(?:Disconnected|Connection reset|Connection closed).*?([0-9a-f:.]+)`)
)

// ParseSSHLine parsea una línea del auth.log de SSH
func ParseSSHLine(line string) *SSHEvent {
	line = strings.TrimSpace(line)

	// Check "Failed password"
	if matches := failedPasswordRe.FindStringSubmatch(line); len(matches) > 0 {
		port, _ := strconv.Atoi(matches[3])
		if isValidIP(matches[2]) {
			return &SSHEvent{
				EventType: "failed_password",
				Username:  matches[1],
				IP:        matches[2],
				Port:      port,
			}
		}
	}

	// Check "Invalid user"
	if matches := invalidUserRe.FindStringSubmatch(line); len(matches) > 0 {
		if isValidIP(matches[2]) {
			return &SSHEvent{
				EventType: "invalid_user",
				Username:  matches[1],
				IP:        matches[2],
			}
		}
	}

	// Check "Accepted"
	if matches := acceptedRe.FindStringSubmatch(line); len(matches) > 0 {
		port, _ := strconv.Atoi(matches[3])
		if isValidIP(matches[2]) {
			return &SSHEvent{
				EventType: "accepted",
				Username:  matches[1],
				IP:        matches[2],
				Port:      port,
			}
		}
	}

	// Check kex/handshake anomalies (solo contar si la IP capturada es válida)
	if matches := kexRe.FindStringSubmatch(line); len(matches) > 0 {
		ip := ""
		if len(matches) > 1 {
			ip = matches[1]
		}
		if ip != "" && isValidIP(ip) {
			return &SSHEvent{
				EventType: "kex_error",
				IP:        ip,
			}
		}
		// IP vacía o inválida (ej. un solo carácter): no generar evento para Top Attackers
		return nil
	}

	if matches := disconnectRe.FindStringSubmatch(line); len(matches) > 0 {
		ip := ""
		if len(matches) > 1 {
			ip = matches[1]
		}
		if ip != "" && isValidIP(ip) {
			return &SSHEvent{
				EventType: "disconnect",
				IP:        ip,
			}
		}
		return nil
	}

	return nil
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
