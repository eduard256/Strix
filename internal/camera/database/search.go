package database

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/eduard256/Strix/internal/models"
)

// SearchEngine handles intelligent camera searching
type SearchEngine struct {
	loader *Loader
	logger interface{ Debug(string, ...any); Error(string, error, ...any) }
	mu     sync.RWMutex
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(loader *Loader, logger interface{ Debug(string, ...any); Error(string, error, ...any) }) *SearchEngine {
	return &SearchEngine{
		loader: loader,
		logger: logger,
	}
}

// SearchResult represents a single search result with score
type SearchResult struct {
	Camera *models.Camera
	Score  float64
}

// Search performs intelligent camera search
func (s *SearchEngine) Search(query string, limit int) (*models.CameraSearchResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	// Normalize query
	normalizedQuery := s.normalizeQuery(query)
	tokens := s.tokenizeQuery(normalizedQuery)

	s.logger.Debug("searching cameras", "query", query, "normalized", normalizedQuery, "tokens", tokens)

	// Extract potential brand and model
	brandToken, modelTokens := s.extractBrandModel(tokens)

	// Perform search
	results, err := s.performSearch(brandToken, modelTokens, normalizedQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Expand each camera into individual model entries with model-specific scores
	type ModelResult struct {
		Camera models.Camera
		Score  float64
	}
	var modelResults []ModelResult

	for _, result := range results {
		camera := result.Camera

		// Collect unique models with their scores
		modelScores := make(map[string]float64)
		for _, entry := range camera.Entries {
			for _, model := range entry.Models {
				if model != "" && model != "Other" {
					// Calculate model-specific score
					modelScore := s.calculateModelScore(model, modelTokens, normalizedQuery)
					if modelScore > modelScores[model] {
						modelScores[model] = modelScore
					}
				}
			}
		}

		// Create a separate camera entry for each unique model
		for model, modelScore := range modelScores {
			// Combine brand score with model score
			finalScore := result.Score*0.3 + modelScore*0.7

			expandedCamera := models.Camera{
				Brand:       camera.Brand,
				BrandID:     camera.BrandID,
				Model:       model,
				LastUpdated: camera.LastUpdated,
				Source:      camera.Source,
				Website:     camera.Website,
				Entries:     camera.Entries,
				MatchScore:  finalScore,
			}
			modelResults = append(modelResults, ModelResult{
				Camera: expandedCamera,
				Score:  finalScore,
			})
		}
	}

	// Sort by final score (best matches first)
	sort.Slice(modelResults, func(i, j int) bool {
		return modelResults[i].Score > modelResults[j].Score
	})

	// Apply limit
	if len(modelResults) > limit {
		modelResults = modelResults[:limit]
	}

	// Convert to camera slice
	cameras := make([]models.Camera, len(modelResults))
	for i, result := range modelResults {
		cameras[i] = result.Camera
	}

	return &models.CameraSearchResponse{
		Cameras:  cameras,
		Total:    len(cameras),
		Returned: len(cameras),
	}, nil
}

// normalizeQuery normalizes the search query
func (s *SearchEngine) normalizeQuery(query string) string {
	// Convert to lowercase
	normalized := strings.ToLower(query)

	// Remove multiple spaces
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Remove special characters but keep spaces
	normalized = regexp.MustCompile(`[^a-z0-9\s\-]`).ReplaceAllString(normalized, " ")

	// Trim spaces
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// tokenizeQuery splits query into tokens
func (s *SearchEngine) tokenizeQuery(query string) []string {
	// Split by spaces and filter empty tokens
	tokens := strings.Fields(query)

	var result []string
	for _, token := range tokens {
		if token != "" {
			result = append(result, token)
		}
	}

	return result
}

// extractBrandModel attempts to extract brand and model from tokens
func (s *SearchEngine) extractBrandModel(tokens []string) (string, []string) {
	if len(tokens) == 0 {
		return "", nil
	}

	// First token is likely the brand
	brandToken := tokens[0]

	// Rest are model tokens
	var modelTokens []string
	if len(tokens) > 1 {
		modelTokens = tokens[1:]
	}

	return brandToken, modelTokens
}

// performSearch executes the actual search
func (s *SearchEngine) performSearch(brandToken string, modelTokens []string, fullQuery string) ([]SearchResult, error) {
	var results []SearchResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Get all brands
	brands, err := s.loader.ListBrands()
	if err != nil {
		return nil, err
	}

	// Search in parallel with limited concurrency
	sem := make(chan struct{}, 10) // Limit to 10 concurrent searches

	for _, brandID := range brands {
		wg.Add(1)
		go func(brandID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Calculate brand match score
			brandScore := s.calculateBrandScore(brandID, brandToken)

			// Skip if brand score is too low
			if brandScore < 0.3 {
				return
			}

			// Load brand data
			camera, err := s.loader.LoadBrand(brandID)
			if err != nil {
				s.logger.Error("failed to load brand", err, "brand", brandID)
				return
			}

			// Calculate model scores for entries
			maxModelScore := 0.0
			for _, entry := range camera.Entries {
				for _, model := range entry.Models {
					modelScore := s.calculateModelScore(model, modelTokens, fullQuery)
					if modelScore > maxModelScore {
						maxModelScore = modelScore
					}
				}
			}

			// Calculate final score
			finalScore := s.calculateFinalScore(brandScore, maxModelScore)

			// Add to results if score is high enough
			if finalScore >= 0.3 {
				mu.Lock()
				results = append(results, SearchResult{
					Camera: camera,
					Score:  finalScore,
				})
				mu.Unlock()
			}
		}(brandID)
	}

	wg.Wait()
	return results, nil
}

// calculateBrandScore calculates how well a brand matches
func (s *SearchEngine) calculateBrandScore(brandID, brandToken string) float64 {
	brandID = strings.ToLower(brandID)
	brandToken = strings.ToLower(brandToken)

	// Exact match
	if brandID == brandToken {
		return 1.0
	}

	// Remove hyphens for comparison
	brandIDClean := strings.ReplaceAll(brandID, "-", "")
	brandTokenClean := strings.ReplaceAll(brandToken, "-", "")

	if brandIDClean == brandTokenClean {
		return 0.95
	}

	// Check if brand starts with token
	if strings.HasPrefix(brandID, brandToken) || strings.HasPrefix(brandIDClean, brandTokenClean) {
		return 0.85
	}

	// Check if token is contained in brand
	if strings.Contains(brandID, brandToken) || strings.Contains(brandIDClean, brandTokenClean) {
		return 0.75
	}

	// Fuzzy match
	if fuzzy.Match(brandToken, brandID) {
		return 0.6
	}

	// Calculate similarity
	similarity := calculateSimilarity(brandID, brandToken)
	return similarity * 0.5
}

// calculateModelScore calculates how well a model matches
func (s *SearchEngine) calculateModelScore(model string, modelTokens []string, fullQuery string) float64 {
	model = strings.ToLower(model)
	fullQuery = strings.ToLower(fullQuery)

	// Check if full query matches the model
	if model == fullQuery {
		return 1.0
	}

	// Check if model contains all tokens
	modelNormalized := s.normalizeQuery(model)
	allTokensFound := true
	tokenMatchScore := 0.0

	for _, token := range modelTokens {
		if strings.Contains(modelNormalized, token) {
			tokenMatchScore += 0.2
		} else {
			allTokensFound = false
		}
	}

	if allTokensFound && len(modelTokens) > 0 {
		return 0.8 + tokenMatchScore/float64(len(modelTokens))*0.2
	}

	// Fuzzy match on full model
	modelCombined := strings.Join(modelTokens, "")
	if fuzzy.Match(modelCombined, modelNormalized) {
		return 0.6
	}

	// Calculate similarity
	similarity := calculateSimilarity(modelNormalized, strings.Join(modelTokens, " "))
	return similarity * 0.5
}

// calculateFinalScore combines brand and model scores
func (s *SearchEngine) calculateFinalScore(brandScore, modelScore float64) float64 {
	// If we have both brand and model matches
	if brandScore > 0 && modelScore > 0 {
		// Weighted average: brand 30%, model 70%
		return brandScore*0.3 + modelScore*0.7
	}

	// If only brand matches
	if brandScore > 0 {
		return brandScore * 0.5
	}

	// If only model matches
	return modelScore * 0.5
}

// SearchByModel searches for cameras by model name with fuzzy matching
func (s *SearchEngine) SearchByModel(modelName string, similarityThreshold float64, limit int) ([]models.Camera, error) {
	if similarityThreshold <= 0 {
		similarityThreshold = 0.8
	}
	if limit <= 0 {
		limit = 6
	}

	normalizedModel := s.normalizeQuery(modelName)
	var results []SearchResult

	// Search through all brands
	cameras, err := s.loader.StreamingSearch(func(camera *models.Camera) bool {
		maxScore := 0.0
		for _, entry := range camera.Entries {
			for _, model := range entry.Models {
				normalizedEntryModel := s.normalizeQuery(model)
				similarity := calculateSimilarity(normalizedModel, normalizedEntryModel)

				// Also check fuzzy match
				if fuzzy.Match(normalizedModel, normalizedEntryModel) {
					if similarity < 0.7 {
						similarity = 0.7
					}
				}

				if similarity > maxScore {
					maxScore = similarity
				}
			}
		}

		if maxScore >= similarityThreshold {
			camera.MatchScore = maxScore
			return true
		}
		return false
	})

	if err != nil {
		return nil, err
	}

	// Convert to SearchResult for sorting
	for _, camera := range cameras {
		results = append(results, SearchResult{
			Camera: camera,
			Score:  camera.MatchScore,
		})
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	// Convert back to Camera slice
	var finalCameras []models.Camera
	for _, result := range results {
		finalCameras = append(finalCameras, *result.Camera)
	}

	return finalCameras, nil
}