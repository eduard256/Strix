package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eduard256/Strix/internal/models"
)

// Loader handles efficient loading of camera database
type Loader struct {
	brandsPath     string
	patternsPath   string
	parametersPath string
	brandsCache    map[string]*models.Camera
	patternsCache  []models.StreamPattern
	paramsCache    []string
	mu             sync.RWMutex
	logger         interface{ Debug(string, ...any); Error(string, error, ...any) }
}

// NewLoader creates a new database loader
func NewLoader(brandsPath, patternsPath, parametersPath string, logger interface{ Debug(string, ...any); Error(string, error, ...any) }) *Loader {
	return &Loader{
		brandsPath:     brandsPath,
		patternsPath:   patternsPath,
		parametersPath: parametersPath,
		brandsCache:    make(map[string]*models.Camera),
		logger:         logger,
	}
}

// LoadBrand loads a specific brand's camera data
func (l *Loader) LoadBrand(brandID string) (*models.Camera, error) {
	l.mu.RLock()
	if cached, ok := l.brandsCache[brandID]; ok {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Load from file
	filePath := filepath.Join(l.brandsPath, brandID+".json")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("brand %s not found", brandID)
		}
		return nil, fmt.Errorf("failed to open brand file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var camera models.Camera
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&camera); err != nil {
		return nil, fmt.Errorf("failed to decode brand data: %w", err)
	}

	// Cache the result
	l.mu.Lock()
	l.brandsCache[brandID] = &camera
	l.mu.Unlock()

	return &camera, nil
}

// ListBrands returns all available brand IDs
func (l *Loader) ListBrands() ([]string, error) {
	files, err := os.ReadDir(l.brandsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read brands directory: %w", err)
	}

	var brands []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			// Skip index files
			if file.Name() == "index.json" || file.Name() == "indexa.json" {
				continue
			}
			brandID := strings.TrimSuffix(file.Name(), ".json")
			brands = append(brands, brandID)
		}
	}

	return brands, nil
}

// LoadPopularPatterns loads popular stream patterns
func (l *Loader) LoadPopularPatterns() ([]models.StreamPattern, error) {
	l.mu.RLock()
	if l.patternsCache != nil {
		patterns := l.patternsCache
		l.mu.RUnlock()
		return patterns, nil
	}
	l.mu.RUnlock()

	file, err := os.Open(l.patternsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open patterns file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var patterns []models.StreamPattern
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&patterns); err != nil {
		return nil, fmt.Errorf("failed to decode patterns: %w", err)
	}

	l.mu.Lock()
	l.patternsCache = patterns
	l.mu.Unlock()

	return patterns, nil
}

// LoadQueryParameters loads supported query parameters
func (l *Loader) LoadQueryParameters() ([]string, error) {
	l.mu.RLock()
	if l.paramsCache != nil {
		params := l.paramsCache
		l.mu.RUnlock()
		return params, nil
	}
	l.mu.RUnlock()

	file, err := os.Open(l.parametersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open parameters file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var params []string
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&params); err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	l.mu.Lock()
	l.paramsCache = params
	l.mu.Unlock()

	return params, nil
}

// StreamingSearch performs memory-efficient search across all brands
func (l *Loader) StreamingSearch(searchFunc func(*models.Camera) bool) ([]*models.Camera, error) {
	files, err := os.ReadDir(l.brandsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read brands directory: %w", err)
	}

	var results []*models.Camera
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Skip index.json as it contains brand list, not camera data
		if file.Name() == "index.json" || file.Name() == "indexa.json" {
			continue
		}

		filePath := filepath.Join(l.brandsPath, file.Name())
		camera, err := l.loadCameraFromFile(filePath)
		if err != nil {
			l.logger.Error("failed to load camera file", err, "file", file.Name())
			continue
		}

		if searchFunc(camera) {
			results = append(results, camera)
		}
	}

	return results, nil
}

// loadCameraFromFile loads a camera from a file without caching
func (l *Loader) loadCameraFromFile(filePath string) (*models.Camera, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var camera models.Camera
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&camera); err != nil {
		return nil, err
	}

	return &camera, nil
}

// GetEntriesForModels returns all entries for specific models with similarity threshold
func (l *Loader) GetEntriesForModels(modelNames []string, similarityThreshold float64) ([]models.CameraEntry, error) {
	entriesMap := make(map[string]models.CameraEntry)

	for _, modelName := range modelNames {
		// Search for similar models across all brands
		cameras, err := l.StreamingSearch(func(camera *models.Camera) bool {
			for _, entry := range camera.Entries {
				for _, model := range entry.Models {
					similarity := calculateSimilarity(modelName, model)
					if similarity >= similarityThreshold {
						return true
					}
				}
			}
			return false
		})

		if err != nil {
			return nil, err
		}

		// Collect unique entries
		for _, camera := range cameras {
			for _, entry := range camera.Entries {
				for _, model := range entry.Models {
					similarity := calculateSimilarity(modelName, model)
					if similarity >= similarityThreshold {
						// Create unique key for deduplication
						key := fmt.Sprintf("%s://%d/%s", entry.Protocol, entry.Port, entry.URL)
						entriesMap[key] = entry
					}
				}
			}
		}
	}

	// Convert map to slice
	var entries []models.CameraEntry
	for _, entry := range entriesMap {
		entries = append(entries, entry)
	}

	return entries, nil
}

// calculateSimilarity calculates similarity between two strings (0.0 to 1.0)
func calculateSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if s1 == s2 {
		return 1.0
	}

	// Simple Levenshtein-based similarity
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 1.0
	}

	distance := levenshteinDistance(s1, s2)
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(values ...int) int {
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ClearCache clears the internal caches
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.brandsCache = make(map[string]*models.Camera)
	l.patternsCache = nil
	l.paramsCache = nil
}