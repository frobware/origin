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

package imagequalifier_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/openshift/origin/pkg/image/admission/imagequalifier"
)

func patterns(rules []imagequalifier.Rule) []string {
	names := make([]string, len(rules))

	for i := range rules {
		names[i] = rules[i].Pattern
	}

	return names
}

func testRules(names []string) []imagequalifier.Rule {
	rules := make([]imagequalifier.Rule, len(names))

	for i, name := range names {
		rules[i].Pattern = name
	}

	return rules
}

func makeTestInput(patterns []string, domain string) string {
	var buffer bytes.Buffer

	for i := range patterns {
		buffer.WriteString(fmt.Sprintf("%s %s\n", patterns[i], domain))
	}

	return buffer.String()
}

func TestSort(t *testing.T) {
	var testcases = []struct {
		input    []string
		expected []string
	}{{
		input: []string{
			"*",
			"*me",
			"*/*/*:latest",
			"*/*/*",
			"*/*:latest",
			"foo*:latest",
			"repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"repo/busybox:1",
			"repo/busybox:latest",
			"repo/busybox:*",
			"repo/busybox",
			"repo/busy",
			"qwerty/busybox",
			"*/*busy",
			"repo/*",
			"repo/busy*",
			"busybox",
			"*/*",
			"busy",
			"*you",
		},
		expected: []string{
			"*",
			"*/*",
			"*/*/*",
			"repo/*",
			"*/*busy",
			"*me",
			"*you",
			"busy",
			"repo/busy",
			"repo/busy*",
			"busybox",
			"qwerty/busybox",
			"repo/busybox:*",
			"repo/busybox",
			"repo/busybox:1",
			"*/*:latest",
			"*/*/*:latest",
			"repo/busybox:latest",
			"foo*:latest",
			"repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}}

	for i, tc := range testcases {
		rules, err := imagequalifier.ParseInput("", makeTestInput(tc.input, "example.com"))
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", err)
		}
		sorted := patterns(imagequalifier.SortRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			// for i := len(sorted) - 1; i >= 0; i-- {
			// 	t.Errorf("%q", sorted[i])
			// }
			// t.Errorf("\n\n\n")

			// for i := range sorted {
			// 	t.Errorf("%q", sorted[i])
			// }
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}
