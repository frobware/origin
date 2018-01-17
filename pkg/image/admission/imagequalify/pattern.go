package imagequalify

import (
	"strings"
)

type PatternParts struct {
	Depth   int
	Digest  string
	Path    string
	Tag     string
	Pattern string
}

func destructurePattern(pattern string) PatternParts {
	parts := PatternParts{
		Path:    pattern,
		Pattern: pattern,
		Depth:   strings.Count(pattern, "/"),
	}

	if i := strings.IndexRune(pattern, '@'); i != -1 {
		parts.Path = pattern[:i]
		parts.Digest = pattern[i+1:]
	}

	if i := strings.IndexRune(parts.Path, ':'); i != -1 {
		parts.Path, parts.Tag = parts.Path[:i], parts.Path[i+1:]
	}

	return parts
}
