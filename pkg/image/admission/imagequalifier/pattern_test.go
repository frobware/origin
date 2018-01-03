/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imagequalifier

import (
	"reflect"
	"testing"
)

func TestPattern(t *testing.T) {
	var testcases = []struct {
		pattern  string
		expected pattern
	}{{
		pattern: "ccc",
		expected: pattern{
			path: "ccc",
		},
	}, {
		pattern: "library/ccc",
		expected: pattern{
			path: "library/ccc",
		},
	}, {
		pattern: "library/ccc:latest",
		expected: pattern{
			path: "library/ccc",
			tag:  "latest",
		},
	}, {
		pattern: "library/ccc:latest@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: pattern{
			path:   "library/ccc",
			tag:    "latest",
			digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}, {
		pattern: "emacs.io/emacs",
		expected: pattern{
			domain: "emacs.io",
			path:   "emacs",
		},
	}, {
		pattern: "localhost/emacs",
		expected: pattern{
			domain: "localhost",
			path:   "emacs",
		},
	}, {
		pattern: "localhost:5000/emacs",
		expected: pattern{
			domain: "localhost:5000",
			path:   "emacs",
		},
	}, {
		pattern: "localhost:5000/emacs:latest",
		expected: pattern{
			domain: "localhost:5000",
			path:   "emacs",
			tag:    "latest",
		},
	}, {
		pattern: "localhost:5000/emacs:latest@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: pattern{
			domain: "localhost:5000",
			path:   "emacs",
			tag:    "latest",
			digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}, {
		pattern: "foo/nginx",
		expected: pattern{
			path: "foo/nginx",
		},
	}, {
		pattern: "foo.io/nginx",
		expected: pattern{
			domain: "foo.io",
			path:   "nginx",
		},
	}, {
		pattern: "foo.io/nginx@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: pattern{
			domain: "foo.io",
			path:   "nginx",
			digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}, {
		pattern: "nginx@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		expected: pattern{
			path:   "nginx",
			digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}}

	for i, tc := range testcases {
		pattern, err := parsePattern(tc.pattern)

		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if !reflect.DeepEqual(tc.expected, *pattern) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, pattern)
		}
	}
}
