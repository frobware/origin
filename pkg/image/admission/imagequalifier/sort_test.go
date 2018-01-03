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
	"fmt"
	"reflect"
	"sort"
	"strings"
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

func TestSortNoWildcards(t *testing.T) {
	var testcases = []struct {
		input    []string
		expected []string
	}{{
		input:    []string{"ccc"},
		expected: []string{"ccc"},
	}, {
		input:    []string{"ccc", "aaa", "bbb", "zzz"},
		expected: []string{"aaa", "bbb", "ccc", "zzz"},
	}, {
		input:    []string{"ccc", "repo/ccc", "aaa", "zzz"},
		expected: []string{"aaa", "ccc", "repo/ccc", "zzz"},
	}}

	for i, tc := range testcases {
		rules := testRules(tc.input)
		//		sorted := patterns(imagequalifier.SortRules(rules))
		sorted := patterns(imagequalifier.OrderRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortWithWildcards(t *testing.T) {
	var testcases = []struct {
		input    []string
		expected []string
	}{{
		input:    []string{"c", "*/c", "a"},
		expected: []string{"*/c", "a", "c"},
	}, {
		input:    []string{"z", "a", "c", "c/c", "*/*/c"},
		expected: []string{"*/*/c", "c/c", "a", "c", "z"},
	}}

	for i, tc := range testcases {
		rules := testRules(tc.input)
		//		sorted := patterns(imagequalifier.SortRules(rules))
		sorted := patterns(imagequalifier.OrderRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortFoo(t *testing.T) {
	var testcases = []struct {
		input    string
		expected []string
	}{{
		input: `
    busybox                 a.io
    busy                    a.io
    repo/busy               a.io
    repo/busybox            a.io
    repo/busybox:*          b.io
    repo/busybox:v1*        c.io
    repo/busybox:v[7-9]*    d.io`,
		expected: []string{
			"repo/busybox",
			"repo/busy",
			"busybox",
			"busy",
			"repo/busybox:v[7-9]*",
			"repo/busybox:v1*",
			"repo/busybox:*",
		},
	}, {
		input: `
    *        a.io
    */*      b.io
    */*/*    c.io
    */*/*:*  d.io`,
		expected: []string{
			"*/*/*:*",
			"*/*/*",
			"*/*",
			"*",
		},
	}}

	for i, tc := range testcases {
		rules, err := imagequalifier.ParseInput("", tc.input)
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", err)
		}
		sorted := patterns(imagequalifier.OrderRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortBar(t *testing.T) {
	var testcases = []struct {
		input    string
		expected []string
	}{{
		input: `
    busybox                 a.io
    busy                    a.io
    repo/busy               a.io
    repo/busybox            a.io
    repo/busybox:*          b.io
    repo/busybox:v1*        c.io
    repo/busybox:v[7-9]*    d.io
    *        a.io
    *me        a.io
    *you      a.io
    */*      b.io
    */*/*    c.io
    */*/*:*  d.io`,
		expected: []string{
			"repo/busybox",
			"repo/busy",
			"busybox",
			"busy",
			"repo/busybox:v[7-9]*",
			"repo/busybox:v1*",
			"repo/busybox:*",
			"*/*/*:*",
			"*/*/*",
			"*/*",
			"*me",
			"*you",
			"*",
		},
	}}

	for i, tc := range testcases {
		rules, err := imagequalifier.ParseInput("", tc.input)
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", err)
		}
		sorted := patterns(imagequalifier.OrderRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortBar2(t *testing.T) {
	var testcases = []struct {
		input    string
		expected []string
	}{{
		input: `
* a.io
*/*/*:latest a.io
*/*/* a.io
*/*:latest a.io
foo*:latest a.io
    repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff c.io
    repo/busybox:1          b.io
    repo/busybox:latest          b.io
    repo/busybox            a.io
    repo/busy               a.io
    qwerty/busybox          a.io
    */*busy                 a.io
    repo/*                  a.io
    repo/busy*              a.io
    busybox                 a.io
    */*                     a.io
    busy                    a.io
`,
		expected: []string{
			"busy",
			"busybox",
			"repo/*",
			"repo/busy",
			"repo/busy*",
			"repo/busybox",
			"repo/busybox:1",
			"repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}}

	for i, tc := range testcases {
		rules, err := imagequalifier.ParseInput("", tc.input)
		if err != nil {
			t.Fatalf("test #%v: unexpected error: %s", err)
		}
		sorted := patterns(imagequalifier.SortRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			for i := len(sorted); i != 0; i-- {
				t.Errorf("%q", sorted[i-1])
			}
			// for i := range sorted {
			// 	t.Errorf("%q", sorted[i])
			// }
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortXXX(t *testing.T) {
	rules := []string{"", "*", "busybox", "*/*"}

	sort.Slice(rules, func(i, j int) bool {
		x := strings.Count(rules[i], "*")
		y := strings.Count(rules[j], "*")
		if x != y {
			return x < y
		}
		return rules[i] < rules[j]
	})
	fmt.Println(rules)
}
