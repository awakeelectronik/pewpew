package firewall

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// UFWBackend implementa operaciones de firewall con UFW
type UFWBackend struct {
	mu sync.Mutex
}

// NewUFWBackend crea nuevo backend UFW
func NewUFWBackend() *UFWBackend {
	return &UFWBackend{}
}

// Ban bloquea una IP
func (u *UFWBackend) Ban(ip string) error {
	if !isValidIP(ip) {
		return fmt.Errorf("invalid IP: %s", ip)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	cmd := exec.Command("ufw", "deny", "from", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("UFW ban failed: %s", string(output))
		return err
	}

	log.Printf("IP %s banned via UFW", ip)
	return nil
}

// Unban desbloquea una IP
func (u *UFWBackend) Unban(ip string) error {
	if !isValidIP(ip) {
		return fmt.Errorf("invalid IP: %s", ip)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	cmd := exec.Command("ufw", "delete", "deny", "from", ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("UFW unban failed: %s", string(output))
		return err
	}

	log.Printf("IP %s unbanned via UFW", ip)
	return nil
}

// IsBanned verifica si una IP está bloqueada
func (u *UFWBackend) IsBanned(ip string) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	cmd := exec.Command("ufw", "status", "numbered")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.Contains(string(output), "deny "+ip), nil
}

// ufwPath devuelve el ejecutable de ufw (PATH o /usr/sbin/ufw). Vacío si no existe.
func ufwPath() string {
	if path, err := exec.LookPath("ufw"); err == nil {
		return path
	}
	if _, err := os.Stat("/usr/sbin/ufw"); err == nil {
		return "/usr/sbin/ufw"
	}
	return ""
}

// IsAvailable verifica si UFW está disponible (incluye /usr/sbin cuando PATH no lo tiene, p. ej. systemd)
func (u *UFWBackend) IsAvailable() bool {
	return ufwPath() != ""
}

// PortProtoKey devuelve la clave "port/proto" para comparar con reglas (p. ej. "5000/tcp").
func PortProtoKey(port int, proto string) string {
	p := strings.ToLower(proto)
	if p != "tcp" && p != "udp" {
		p = "tcp"
	}
	return strconv.Itoa(port) + "/" + p
}

// NUM/tcp o NUM/udp + action (ej. "22/tcp ALLOW", "443/tcp DENY")
var ufwRuleLineRe = regexp.MustCompile(`(\d+)/(tcp|udp).*?(allow|deny|reject|limit)`)
// Puerto solo + action (ej. "5000    DENY", "3005 (v6) DENY") — UFW sin /tcp en la regla
var ufwRulePortOnlyRe = regexp.MustCompile(`(?i)^\s*(\d+)\s+(?:\(v6\)\s+)?(allow|deny|reject|limit)\s+`)

// PortRules obtiene las reglas de puertos de UFW (qué está ALLOW o DENY).
func (u *UFWBackend) PortRules() (allowed map[string]bool, denied map[string]bool, err error) {
	allowed = make(map[string]bool)
	denied = make(map[string]bool)

	path := ufwPath()
	if path == "" {
		return nil, nil, fmt.Errorf("ufw not found")
	}
	cmd := exec.Command(path, "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		// Primero: formato "5000/tcp DENY"
		matches := ufwRuleLineRe.FindStringSubmatch(lineLower)
		if len(matches) == 4 {
			portStr, proto, action := matches[1], matches[2], matches[3]
			port, _ := strconv.Atoi(portStr)
			if port != 0 {
				key := PortProtoKey(port, proto)
				switch action {
				case "allow", "limit":
					allowed[key] = true
				case "deny", "reject":
					denied[key] = true
				}
			}
			continue
		}
		// Segundo: formato "5000    DENY" (puerto sin /tcp ni /udp) — aplicar a tcp y udp
		matchesPortOnly := ufwRulePortOnlyRe.FindStringSubmatch(line)
		if len(matchesPortOnly) == 3 {
			portStr, action := matchesPortOnly[1], strings.ToLower(matchesPortOnly[2])
			port, _ := strconv.Atoi(portStr)
			if port != 0 {
				switch action {
				case "allow", "limit":
					allowed[PortProtoKey(port, "tcp")] = true
					allowed[PortProtoKey(port, "udp")] = true
				case "deny", "reject":
					denied[PortProtoKey(port, "tcp")] = true
					denied[PortProtoKey(port, "udp")] = true
				}
			}
		}
	}
	return allowed, denied, nil
}

// IsPortBlockedByUFW indica si el puerto/proto está explícitamente DENY/REJECT en UFW.
// Si UFW no está disponible o falla el parseo, devuelve false (no asumimos bloqueado).
func IsPortBlockedByUFW(port int, proto string) bool {
	u := NewUFWBackend()
	if !u.IsAvailable() {
		return false
	}
	_, denied, err := u.PortRules()
	if err != nil || len(denied) == 0 {
		return false
	}
	key := PortProtoKey(port, proto)
	return denied[key]
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
