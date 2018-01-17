package imagequalify

import (
	"strings"
)

type PatternParts struct {
	Depth  int
	Digest string
	Path   string
	Tag    string
}

func destructurePattern(s string) PatternParts {
	parts := PatternParts{
		Path:  s,
		Depth: strings.Count(s, "/"),
	}

	if i := strings.IndexRune(s, '@'); i != -1 {
		parts.Path = s[:i]
		parts.Digest = s[i+1:]
	}

	if i := strings.IndexRune(parts.Path, ':'); i != -1 {
		parts.Path, parts.Tag = parts.Path[:i], parts.Path[i+1:]
	}

	return parts
}
