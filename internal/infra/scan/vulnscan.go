package scan

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"pewpew/internal/domain"
)

// RunVulnerabilityScan realiza checks rapidos de red sobre puertos abiertos.
func RunVulnerabilityScan(ports []*domain.OpenPort) []*domain.VulnerabilityFinding {
	findings := make([]*domain.VulnerabilityFinding, 0, 16)
	timeout := 1200 * time.Millisecond

	for _, p := range ports {
		if !p.Externally {
			continue
		}
		target := net.JoinHostPort("127.0.0.1", strconv.Itoa(p.Port))
		conn, err := net.DialTimeout("tcp", target, timeout)
		if err != nil {
			continue
		}
		_ = conn.Close()

		switch p.Port {
		case 22:
			findings = append(findings, &domain.VulnerabilityFinding{
				ID:             fmt.Sprintf("ssh-%d", p.Port),
				Severity:       "medium",
				Category:       "ssh",
				Target:         p.Address + ":" + strconv.Itoa(p.Port),
				Evidence:       "SSH service reachable from all interfaces",
				Recommendation: "Use key-only auth, fail2ban and restrict access by firewall/IP allowlist",
			})
		case 2375:
			findings = append(findings, &domain.VulnerabilityFinding{
				ID:             "docker-2375",
				Severity:       "critical",
				Category:       "docker",
				Target:         p.Address + ":2375",
				Evidence:       "Docker daemon plaintext API exposed",
				Recommendation: "Disable exposed Docker TCP API or protect with TLS and firewall restrictions",
			})
		case 3306, 5432, 6379, 27017:
			findings = append(findings, &domain.VulnerabilityFinding{
				ID:             fmt.Sprintf("db-%d", p.Port),
				Severity:       "high",
				Category:       "database",
				Target:         p.Address + ":" + strconv.Itoa(p.Port),
				Evidence:       "Database port reachable externally",
				Recommendation: "Bind DB to localhost/private network and enforce firewall deny from internet",
			})
		case 9200:
			findings = append(findings, &domain.VulnerabilityFinding{
				ID:             "es-9200",
				Severity:       "high",
				Category:       "elasticsearch",
				Target:         p.Address + ":9200",
				Evidence:       "Elasticsearch HTTP API exposed",
				Recommendation: "Require authentication and restrict network access to trusted hosts",
			})
		}
	}

	return findings
}
