package probe

import (
	"database/sql"
	"strings"
)

// LookupOUI returns vendor name for MAC address from SQLite oui table.
// MAC format: "C0:56:E3:AA:BB:CC" -> prefix "C0:56:E3"
func LookupOUI(db *sql.DB, mac string) string {
	if len(mac) < 8 {
		return ""
	}

	prefix := strings.ToUpper(mac[:8])
	prefix = strings.ReplaceAll(prefix, "-", ":")

	var brand string
	_ = db.QueryRow("SELECT brand FROM oui WHERE prefix = ?", prefix).Scan(&brand)
	return brand
}
