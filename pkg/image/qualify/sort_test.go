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

package qualify_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/openshift/origin/pkg/image/qualify"
)

func patterns(rules []qualify.Rule) []string {
	names := make([]string, len(rules))

	for i := range rules {
		names[i] = rules[i].Pattern
	}

	return names
}

func addDomain(input string) string {
	var bb bytes.Buffer

	for i, line := range strings.Split(input, "\n") {
		for j, word := range strings.Fields(line) {
			bb.WriteString(fmt.Sprintf("%s domain-%v.com\n", word, i+j))
		}
	}

	return bb.String()
}

func TestSort(t *testing.T) {
	// Note: priority ordering is lowest..highest. And entries
	// with a wildcard character '*' always sort lower than
	// entries without.

	var testcases = []struct {
		description string
		input       string
		expected    string
	}{{
		description: "default order is collating sequence",
		input:       "c b a",
		expected:    "a b c",
	}, {
		description: "wildcard default order is collating sequence",
		input:       "*/*/*@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff */*/* * */*/*:latest */*",
		expected:    "* */* */*/* */*/*:latest */*/*@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}, {
		description: "default order is collating sequence, even for library components",
		input:       "foo/emacs foo/vim emacs vim foo/* */* *",
		expected:    "* */* foo/* emacs foo/emacs foo/vim vim",
	}, {
		description: "wild cards sort lower",
		input:       "a*b abc */* * */*/*",
		expected:    "* */* */*/* a*b abc",
	}, {
		description: "tags sort lower",
		input:       "abc abc:latest abc:1.0 */* *",
		expected:    "* */* abc abc:1.0 abc:latest",
	}, {
		description: "digests sort lower",
		input:       "abc abc@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff */* *",
		expected:    "* */* abc abc@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}, {
		description: "image references with library components sort lower",
		input:       "foo/emacs foo/vim emacs vim */* *",
		expected:    "* */* emacs foo/emacs foo/vim vim",
	}, {
		description: "wildcard library references sort lower",
		input:       "foo/emacs */vim emacs vim */* *",
		expected:    "* */* */vim emacs foo/emacs vim",
	}, {
		description: "wildcard tags sort lower",
		input:       "foo/emacs:* */vim foo/emacs emacs vim */* *",
		expected:    "* */* */vim foo/emacs:* emacs foo/emacs vim",
	}, {
		description: "wildcard libraries",
		input:       "*me *you * */* */* */*/*",
		expected:    "* */* */* */*/* *me *you",
	}, {
		description: "library references",
		input: `repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
                        repo/busybox:1
                        repo/busybox:latest
                        repo/busybox:*
                        repo/busybox
                        repo/busy
                        qwerty/busybox
                        */*busy
                        repo/*
                        repo/busy*
                        busybox`,
		expected: `*/*busy
                           repo/*
                           repo/busy*
                           repo/busybox:*
                           busybox
                           qwerty/busybox
                           repo/busy
                           repo/busybox
                           repo/busybox:1
                           repo/busybox:latest
                           repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`,
	}}

	for i, tc := range testcases {
		rules, err := qualify.ParseRules(addDomain(tc.input))
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", i, err)
		}

		expected, err := qualify.ParseRules(addDomain(tc.expected))
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", err)
		}

		sorted := patterns(qualify.SortRules(rules))

		if !reflect.DeepEqual(patterns(expected), sorted) {
			t.Errorf("test #%v: %s, expected %#v, got %#v", i, tc.description, patterns(expected), sorted)

		}
	}
}

// func TestSort(t *testing.T) {
//	var testcases = []struct {
//		input    []string
//		expected []string
//	}{{
//		input: []string{
//			"*",
//			"*me",
//			"*/*/*:latest",
//			"*/*/*",
//			"*/*:latest",
//			"foo*:latest",
//			"repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
//			"repo/busybox:1",
//			"repo/busybox:latest",
//			"repo/busybox:*",
//			"repo/busybox",
//			"repo/busy",
//			"qwerty/busybox",
//			"*/*busy",
//			"repo/*",
//			"repo/busy*",
//			"busybox",
//			"*/*",
//			"busy",
//			"*you",
//		},
//		expected: []string{
//			"*",
//			"*/*",
//			"*/*/*",
//			"repo/*",
//			"*/*busy",
//			"*me",
//			"*you",
//			"sort.com/*/*you",
//			"busy",
//			"repo/busy",
//			"l/busybox:*@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
//			"repo/busy*",
//			"busybox",
//			"qwerty/busybox",
//			"repo/busybox:*",
//			"repo/busybox",
//			"repo/busybox:1",
//			"*/*:latest",
//			"*/*/*:latest",
//			"repo/busybox:latest",
//			"foo*:latest",
//			"repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
//		},
//	}}

//	for _, tc := range testcases {
//		rules, err := qualify.ParseRules(inputToRule(tc.input))
//		if err != nil {
//			t.Fatalf("test #%v: unexpected error: %s", err)
//		}
//		sorted := patterns(qualify.SortRules(rules))

//		if !reflect.DeepEqual(tc.expected, sorted) {
//			// for i := len(sorted) - 1; i >= 0; i-- {
//			//	t.Errorf("%q", sorted[i])
//			// }
//			// t.Errorf("\n\n\n")

//			// for i := range sorted {
//			//	t.Errorf("%q", sorted[i])
//			// }
//			// t.Errorf("test #%v: expected %#v, got %#v", i, tc.expect
//			//				ed, sorted)
//		}
//	}
// }
