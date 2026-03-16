package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// OUIDatabase provides MAC address prefix to vendor name lookup.
// Data is loaded from a JSON file containing camera/surveillance vendor OUI prefixes.
type OUIDatabase struct {
	data map[string]string // "C0:56:E3" -> "Hikvision"
	mu   sync.RWMutex
}

// NewOUIDatabase creates an empty OUI database.
func NewOUIDatabase() *OUIDatabase {
	return &OUIDatabase{
		data: make(map[string]string),
	}
}

// LoadFromFile loads OUI data from a JSON file.
// Expected format: {"C0:56:E3": "Hikvision", "54:EF:44": "Lumi/Aqara", ...}
func (db *OUIDatabase) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open OUI database: %w", err)
	}
	defer file.Close()

	var data map[string]string
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode OUI database: %w", err)
	}

	// Normalize all keys to uppercase
	normalized := make(map[string]string, len(data))
	for k, v := range data {
		normalized[strings.ToUpper(k)] = v
	}

	db.mu.Lock()
	db.data = normalized
	db.mu.Unlock()

	return nil
}

// LookupVendor returns the vendor name for a given MAC address.
// MAC can be in any format: "C0:56:E3:AA:BB:CC", "c0:56:e3:aa:bb:cc", "C0-56-E3-AA-BB-CC".
// Returns empty string if not found.
func (db *OUIDatabase) LookupVendor(mac string) string {
	if len(mac) < 8 {
		return ""
	}

	// Normalize: uppercase and replace dashes with colons
	prefix := strings.ToUpper(mac[:8])
	prefix = strings.ReplaceAll(prefix, "-", ":")

	db.mu.RLock()
	vendor := db.data[prefix]
	db.mu.RUnlock()

	return vendor
}

// Size returns the number of entries in the database.
func (db *OUIDatabase) Size() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.data)
}
