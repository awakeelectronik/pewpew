package firewall

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
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
	defer tu.mu.Unlock()

	cmd := exec.Command("sudo", "ufw", "deny", "from", ip)
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

	cmd := exec.Command("sudo", "ufw", "delete", "deny", "from", ip)
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

	cmd := exec.Command("sudo", "ufw", "status", "numbered")
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

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
