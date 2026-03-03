package scan

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"pewpew/internal/domain"
	"pewpew/internal/infra/firewall"
)

// DetectOpenPorts obtiene puertos en escucha usando ss.
// Los puertos que UFW tiene en DENY/REJECT no se consideran expuestos (risk low, "Blocked by UFW").
func DetectOpenPorts() ([]*domain.OpenPort, error) {
	cmd := exec.Command("ss", "-lntupH")
	out, err := cmd.Output()
	if err != nil {
		return []*domain.OpenPort{}, err
	}

	ufwDenied := ufwDeniedPorts() // una sola consulta a ufw status

	scanner := bufio.NewScanner(bytes.NewReader(out))
	ports := make([]*domain.OpenPort, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := fields[0]
		state := fields[1]
		local := fields[4]
		process := ""
		if len(fields) > 6 {
			process = strings.Join(fields[6:], " ")
		}

		address, port := parseAddrPort(local)
		if port == 0 {
			continue
		}
		externally := isExternalBind(address)
		blockedByUFW := ufwDenied[firewall.PortProtoKey(port, proto)]
		if externally && blockedByUFW {
			externally = false
		}
		risk, reason := classifyPortRisk(port, externally)
		if blockedByUFW && isExternalBind(address) {
			reason = "Blocked by UFW"
		}
		ports = append(ports, &domain.OpenPort{
			Proto:      proto,
			Address:    address,
			Port:       port,
			Process:    process,
			State:      state,
			Risk:       risk,
			Reason:     reason,
			Externally: externally,
		})
	}

	return ports, scanner.Err()
}

// ufwDeniedPorts devuelve el conjunto de claves "port/proto" que UFW tiene en DENY/REJECT.
func ufwDeniedPorts() map[string]bool {
	u := firewall.NewUFWBackend()
	if !u.IsAvailable() {
		return nil
	}
	_, denied, err := u.PortRules()
	if err != nil {
		return nil
	}
	return denied
}

func parseAddrPort(input string) (string, int) {
	// [::]:22 or 0.0.0.0:22
	last := strings.LastIndex(input, ":")
	if last <= 0 || last+1 >= len(input) {
		return input, 0
	}
	addr := strings.Trim(input[:last], "[]")
	p, _ := strconv.Atoi(input[last+1:])
	return addr, p
}

func isExternalBind(addr string) bool {
	return addr == "0.0.0.0" || addr == "::" || addr == "*"
}

func classifyPortRisk(port int, external bool) (string, string) {
	if !external {
		return "low", "Bound to local interface"
	}
	switch port {
	case 22, 2375, 3306, 5432, 6379, 27017, 9200:
		return "high", "Sensitive service exposed externally"
	case 80, 443:
		return "medium", "Public web service exposed"
	default:
		return "medium", "Port exposed on all interfaces"
	}
}
