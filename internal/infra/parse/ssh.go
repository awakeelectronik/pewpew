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

	return nil
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
