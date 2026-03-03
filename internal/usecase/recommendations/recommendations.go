package recommendations

import (
	"os/exec"
	"strings"

	"pewpew/internal/domain"
)

// Build genera recomendaciones de hardening en base al host.
func Build(ports []*domain.OpenPort, firewallBackend string) []*domain.Recommendation {
	out := make([]*domain.Recommendation, 0, 8)

	if firewallBackend == "none" {
		out = append(out, &domain.Recommendation{
			ID:       "firewall",
			Severity: "critical",
			Title:    "Firewall backend not detected",
			Details:  "No active firewall backend detected by PewPew",
			Command:  "sudo apt install ufw && sudo ufw enable",
		})
	} else {
		out = append(out, &domain.Recommendation{
			ID:       "firewall-ok",
			Severity: "ok",
			Title:    "Firewall backend detected",
			Details:  "Detected backend: " + firewallBackend,
		})
	}

	if hasExternalSSH(ports) {
		out = append(out, &domain.Recommendation{
			ID:       "ssh-exposed",
			Severity: "warn",
			Title:    "SSH exposed to internet",
			Details:  "Port 22 is reachable externally; brute-force noise is expected",
			Command:  "sudo ufw limit 22/tcp",
		})
	}

	if !isFail2banActive() {
		out = append(out, &domain.Recommendation{
			ID:       "fail2ban",
			Severity: "warn",
			Title:    "Fail2ban not active",
			Details:  "Fail2ban helps auto-ban repetitive SSH attacks",
			Command:  "sudo apt install fail2ban && sudo systemctl enable --now fail2ban",
		})
	}

	if !hasSensitiveExposed(ports) {
		out = append(out, &domain.Recommendation{
			ID:       "exposure-ok",
			Severity: "ok",
			Title:    "No sensitive public ports detected",
			Details:  "No critical DB/Docker ports detected as externally reachable",
		})
	}

	return out
}

func isFail2banActive() bool {
	cmd := exec.Command("systemctl", "is-active", "fail2ban")
	b, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "active"
}

func hasExternalSSH(ports []*domain.OpenPort) bool {
	for _, p := range ports {
		if p.Port == 22 && p.Externally {
			return true
		}
	}
	return false
}

func hasSensitiveExposed(ports []*domain.OpenPort) bool {
	for _, p := range ports {
		if !p.Externally {
			continue
		}
		switch p.Port {
		case 2375, 3306, 5432, 6379, 27017, 9200:
			return true
		}
	}
	return false
}
