package imagequalify

import "github.com/openshift/origin/pkg/image/admission/imagequalify/api"

// ByPatternPriority sorts a slice by pattern depth, in ascending
// order.
type ByPatternPriority []api.ImageQualifyRule

func (x ByPatternPriority) Len() int {
	return len(x)
}

func (x ByPatternPriority) Less(i, j int) bool {
	return x[i].Pattern > x[j].Pattern
}

func (x ByPatternPriority) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}
