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
	"reflect"
	"sort"
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
		input:    nil,
		expected: []string{},
	}, {
		input:    []string{},
		expected: []string{},
	}, {
		input:    []string{"ccc"},
		expected: []string{"ccc"},
	}, {
		input:    []string{"ccc", "aaa", "bbb", "zzz"},
		expected: []string{"aaa", "bbb", "ccc", "zzz"},
	}, {
		input:    []string{"ccc", "aaa", "repo/bbb", "zzz"},
		expected: []string{"aaa", "bbb", "repo/bbb", "zzz"},
	}, {
		input:    []string{"ccc", "aaa", "repo/bbb", "domain.io/emacs"},
		expected: []string{"aaa", "bbb", "repo/bbb", "zzz"},
	}, {
		input:    []string{"ccc", "aaa", "repo/bbb", "domain.io/lib/zzz:latest"},
		expected: []string{"aaa", "bbb", "repo/bbb", "zzz"},
	}, {
		input:    []string{"ccc", "aaa", "repo/bbb", "domain.io/lib/zzz:latest@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		expected: []string{"aaa", "bbb", "repo/bbb", "zzz"},
	}}

	for i, tc := range testcases {
		rules := testRules(tc.input)
		sorted := patterns(imagequalifier.SortRules(rules))

		if !reflect.DeepEqual(tc.expected, sorted) {
			t.Errorf("test #%v: expected %#v, got %#v", i, tc.expected, sorted)
		}
	}
}

func TestSortExplicitRulesSortFirst(t *testing.T) {
	rules := []imagequalifier.Rule{
		imagequalifier.Rule{Pattern: "bbb"},
		imagequalifier.Rule{Pattern: "aaa"},
		imagequalifier.Rule{Pattern: "ccc"},
	}

	sorted := imagequalifier.SortRules(rules)

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Pattern < rules[j].Pattern
	})

	if !reflect.DeepEqual(rules, sorted) {
		t.Errorf("expected %#v, got %#v", rules, sorted)
	}
}
