package firewall

import (
	"fmt"
	"log"
	"net"
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

// IsAvailable verifica si UFW está disponible
func (u *UFWBackend) IsAvailable() bool {
	cmd := exec.Command("which", "ufw")
	return cmd.Run() == nil
}

// PortProtoKey devuelve la clave "port/proto" para comparar con reglas (p. ej. "5000/tcp").
func PortProtoKey(port int, proto string) string {
	p := strings.ToLower(proto)
	if p != "tcp" && p != "udp" {
		p = "tcp"
	}
	return strconv.Itoa(port) + "/" + p
}

var ufwRuleLineRe = regexp.MustCompile(`^\s*(\d+)/(tcp|udp)\s+(ALLOW|DENY|REJECT|LIMIT)\s+`)

// PortRules obtiene las reglas de puertos de UFW (qué está ALLOW o DENY desde Anywhere).
// Útil para no marcar como "expuesto" un puerto que UFW bloquea.
func (u *UFWBackend) PortRules() (allowed map[string]bool, denied map[string]bool, err error) {
	allowed = make(map[string]bool)
	denied = make(map[string]bool)

	cmd := exec.Command("ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(string(output), "\n")
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "To ") && strings.Contains(trimmed, "Action") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		// Línea de regla: "22/tcp    ALLOW    Anywhere" o "5000/tcp  DENY  Anywhere"
		matches := ufwRuleLineRe.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		portStr, proto, action := matches[1], strings.ToLower(matches[2]), matches[3]
		port, _ := strconv.Atoi(portStr)
		if port == 0 {
			continue
		}
		key := PortProtoKey(port, proto)
		switch action {
		case "ALLOW", "LIMIT":
			allowed[key] = true
		case "DENY", "REJECT":
			denied[key] = true
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
