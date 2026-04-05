package camdb

import (
	"database/sql"
	"strings"
)

type Result struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchAll returns all presets + all brands, no models
func SearchAll(db *sql.DB) ([]Result, error) {
	var results []Result

	rows, err := db.Query("SELECT preset_id, name FROM presets ORDER BY preset_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		results = append(results, Result{Type: "preset", ID: "p:" + id, Name: name})
	}

	rows, err = db.Query("SELECT brand_id, brand FROM brands ORDER BY brand LIMIT ?", 50-len(results))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		results = append(results, Result{Type: "brand", ID: "b:" + id, Name: name})
	}

	return results, nil
}

// SearchQuery searches presets, brands, models by query string (limit 50 total).
// Supports: "model", "brand model", "model brand" -- each word matches independently.
func SearchQuery(db *sql.DB, q string) ([]Result, error) {
	var results []Result
	like := "%" + q + "%"

	// presets
	rows, err := db.Query(
		"SELECT preset_id, name FROM presets WHERE preset_id LIKE ? OR name LIKE ? ORDER BY preset_id",
		like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		results = append(results, Result{Type: "preset", ID: "p:" + id, Name: name})
	}

	// brands
	rows, err = db.Query(
		"SELECT brand_id, brand FROM brands WHERE brand_id LIKE ? OR brand LIKE ? ORDER BY brand LIMIT ?",
		like, like, 50-len(results),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		results = append(results, Result{Type: "brand", ID: "b:" + id, Name: name})
	}

	if len(results) >= 50 {
		return results, nil
	}

	// models -- each word must match brand or model
	words := strings.Fields(q)
	where := ""
	args := make([]any, 0, len(words)+1)
	for i, w := range words {
		if i > 0 {
			where += " AND "
		}
		where += "(b.brand LIKE ? OR b.brand_id LIKE ? OR sm.model LIKE ?)"
		p := "%" + w + "%"
		args = append(args, p, p, p)
	}
	args = append(args, 50-len(results))

	rows, err = db.Query(
		`SELECT DISTINCT b.brand_id, b.brand, sm.model
		FROM stream_models sm
		JOIN streams s ON s.id = sm.stream_id
		JOIN brands b ON b.brand_id = s.brand_id
		WHERE `+where+`
		ORDER BY b.brand, sm.model
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var brandID, brand, model string
		if err = rows.Scan(&brandID, &brand, &model); err != nil {
			return nil, err
		}
		results = append(results, Result{
			Type: "model",
			ID:   "m:" + brandID + ":" + model,
			Name: brand + ": " + model,
		})
	}

	return results, nil
}
