package imagequalify

import "github.com/openshift/origin/pkg/image/admission/imagequalify/api"

// ByPatternAscending sorts a slice lexicographically.
type ByPatternAscending []api.ImageQualifyRule

func (x ByPatternAscending) Len() int {
	return len(x)
}

func (x ByPatternAscending) Less(i, j int) bool {
	return x[i].Pattern > x[j].Pattern
}

func (x ByPatternAscending) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}
