package imagequalify_test

import (
	"reflect"
	"testing"

	"github.com/openshift/origin/pkg/image/admission/imagequalify"
)

func TestPatternParse(t *testing.T) {
	var testcases = []struct {
		pattern  string
		expected imagequalify.PatternParts
	}{{
		pattern: "a",
		expected: imagequalify.PatternParts{
			Path: "a",
		},
	}, {
		pattern: "a/b",
		expected: imagequalify.PatternParts{
			Path: "a/b",
		},
	}, {
		pattern: "a:latest",
		expected: imagequalify.PatternParts{
			Path: "a",
			Tag:  "latest",
		},
	}, {
		pattern: "a@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: imagequalify.PatternParts{
			Path:   "a",
			Digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}, {
		pattern: "a:latest@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: imagequalify.PatternParts{
			Path:   "a",
			Tag:    "latest",
			Digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}, {
		pattern: "repo/a:latest@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: imagequalify.PatternParts{
			Path:   "repo/a",
			Tag:    "latest",
			Digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}}

	for i, tc := range testcases {
		actual := imagequalify.DestructurePattern(tc.pattern)
		if !reflect.DeepEqual(tc.expected, actual) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, actual)
		}
	}
}
