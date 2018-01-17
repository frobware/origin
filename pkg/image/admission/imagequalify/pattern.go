package imagequalify

import (
	"strings"
)

type PatternParts struct {
	Path   string
	Tag    string
	Digest string
}

func destructurePattern(s string) PatternParts {
	parts := PatternParts{Path: s}

	if i := strings.IndexRune(s, '@'); i != -1 {
		parts.Path = s[:i]
		parts.Digest = s[i+1:]
	}

	if i := strings.IndexRune(parts.Path, ':'); i != -1 {
		parts.Path, parts.Tag = parts.Path[:i], parts.Path[i+1:]
		if i := strings.IndexRune(parts.Tag, '@'); i != -1 {
			parts.Tag = parts.Tag[:i]
		}
	}

	return parts
}
