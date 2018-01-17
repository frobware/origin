package imagequalify

import (
	"sort"
	"strings"

	"github.com/openshift/origin/pkg/image/admission/imagequalify/api"
)

// ByPatternPathAscending sorts a slice lexicographically.
type ByPatternPathAscending []api.ImageQualifyRule

func (x ByPatternPathAscending) Len() int {
	return len(x)
}

func (x ByPatternPathAscending) Less(i, j int) bool {
	a := destructurePattern(x[i].Pattern)
	b := destructurePattern(x[j].Pattern)
	return a.Path > b.Path
}

func (x ByPatternPathAscending) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

// ByPatternDepth sorts a slice lexicographically.
type ByPatternDepth []api.ImageQualifyRule

func (x ByPatternDepth) Len() int {
	return len(x)
}

func (x ByPatternDepth) Less(i, j int) bool {
	return strings.Count(x[i].Pattern, "/") > strings.Count(x[j].Pattern, "/")
}

func (x ByPatternDepth) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

// ByPatternDigestAscending sorts a slice lexicographically.
type ByPatternDigestAscending []api.ImageQualifyRule

func (x ByPatternDigestAscending) Len() int {
	return len(x)
}

func (x ByPatternDigestAscending) Less(i, j int) bool {
	a := destructurePattern(x[i].Pattern)
	b := destructurePattern(x[j].Pattern)
	return a.Digest > b.Digest
}

func (x ByPatternDigestAscending) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

// ByPatternTagAscending sorts a slice lexicographically.
type ByPatternTagAscending []api.ImageQualifyRule

func (x ByPatternTagAscending) Len() int {
	return len(x)
}

func (x ByPatternTagAscending) Less(i, j int) bool {
	a := destructurePattern(x[i].Pattern)
	b := destructurePattern(x[j].Pattern)
	return a.Tag > b.Tag
}

func (x ByPatternTagAscending) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

type lessFunc func(x, y *PatternParts) bool

type PatternPartsSorter struct {
	rules []api.ImageQualifyRule
	less  []lessFunc
}

// Sort sorts the argument slice according to the less functions
// passed to orderBy.
func (s *PatternPartsSorter) Sort(rules []api.ImageQualifyRule) {
	s.rules = rules
	sort.Sort(s)
}

func (s *PatternPartsSorter) Stable(rules []api.ImageQualifyRule) {
	s.rules = rules
	sort.Stable(s)
}

// orderBy returns a Sorter that sorts using the less functions, in
// order. Call its Sort method to sort the data.
func orderBy(less ...lessFunc) *PatternPartsSorter {
	return &PatternPartsSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (s *PatternPartsSorter) Len() int {
	return len(s.rules)
}

// Swap is part of sort.Interface.
func (s *PatternPartsSorter) Swap(i, j int) {
	s.rules[i], s.rules[j] = s.rules[j], s.rules[i]
}

func (s *PatternPartsSorter) Less(i, j int) bool {
	// p, q := s.rules[i], s.rules[j]

	a := destructurePattern(s.rules[i].Pattern)
	b := destructurePattern(s.rules[j].Pattern)

	// Try all but the last comparison.
	var k int

	// TODO use CompareTo.
	for k = 0; k < len(s.less)-1; k++ {
		less := s.less[k]
		switch {
		case less(&a, &b):
			// p < q, so we have a decision.
			return true
		case less(&b, &a):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}

	return s.less[k](&a, &b)
}
