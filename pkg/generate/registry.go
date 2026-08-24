package generate

import "strings"

// ExtractFunc cleans rawURL (ex. strips ?token=...) and returns a top-level
// YAML section name + key/value to upsert into the config.
// Returns empty section if the URL has nothing to extract -- cleaned URL
// is still used as-is.
type ExtractFunc func(rawURL string) (cleaned, section, key, value string)

var extractors = map[string]ExtractFunc{}

func RegisterExtract(scheme string, fn ExtractFunc) {
	extractors[scheme] = fn
}

func runExtract(rawURL string) (cleaned, section, key, value string) {
	if i := strings.IndexByte(rawURL, ':'); i > 0 {
		if fn := extractors[rawURL[:i]]; fn != nil {
			return fn(rawURL)
		}
	}
	return rawURL, "", "", ""
}
