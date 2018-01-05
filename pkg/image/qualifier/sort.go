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

package qualifier

import (
	"fmt"
	"sort"
)

type lessFunc func(x, y *pattern) bool

type patternSorter struct {
	rules []Rule
	less  []lessFunc
}

// Sort sorts the argument slice according to the less functions
// passed to orderby.
func (ms *patternSorter) Sort(rules []Rule) {
	ms.rules = rules
	sort.Sort(ms)
}

// Orderby returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func orderby(less ...lessFunc) *patternSorter {
	return &patternSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *patternSorter) Len() int {
	return len(ms.rules)
}

// Swap is part of sort.Interface.
func (ms *patternSorter) Swap(i, j int) {
	ms.rules[i], ms.rules[j] = ms.rules[j], ms.rules[i]
}

// Less is part of sort.Interface. It is implemented by looping along
// the less functions until it finds a comparison that is either Less
// or !Less. Note that it can call the less functions twice per call.
// We could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
func (ms *patternSorter) Less(i, j int) bool {
	p, q := ms.rules[i], ms.rules[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p.pattern, q.pattern):
			// p < q, so we have a decision.
			return true
		case less(q.pattern, p.pattern):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}

	return ms.less[k](p.pattern, q.pattern)
}

func globcmp(a, b string) bool {
	if a == "*" && b == "" {
		return false
	}
	if a == "" && b == "*" {
		return false
	}
	return a < b
}

func ruleCompare(a, b Rule) bool {
	if string(a.pattern.digest) < string(b.pattern.digest) {
		return true
	} else if string(b.pattern.digest) < string(a.pattern.digest) {
		return false
	}

	if globcmp(a.pattern.tag, b.pattern.tag) {
		return true
	} else if globcmp(b.pattern.tag, a.pattern.tag) {
		return false
	}

	if globcmp(a.pattern.image, b.pattern.image) {
		return true
	} else if globcmp(b.pattern.image, a.pattern.image) {
		return false
	}

	if globcmp(a.pattern.library, b.pattern.library) {
		return true
	} else if globcmp(b.pattern.library, a.pattern.library) {
		return false
	}

	if globcmp(a.pattern.domain, b.pattern.domain) {
		return true
	} else if globcmp(b.pattern.domain, a.pattern.domain) {
		return false
	}

	return false
}

func sortRules(rules []Rule) []Rule {
	if true {
		digest := func(x, y *pattern) bool {
			return string(x.digest) < string(y.digest)
		}

		tag := func(x, y *pattern) bool {
			return globcmp(x.tag, y.tag)
		}

		image := func(x, y *pattern) bool {
			return globcmp(x.image, y.image)
		}

		library := func(x, y *pattern) bool {
			return globcmp(x.library, y.library)
		}

		domain := func(x, y *pattern) bool {
			return globcmp(x.domain, y.domain)
		}

		path := func(x, y *pattern) bool {
			return globcmp(x.path, y.path)
		}

		orderby(digest, tag, image, library, path, domain).Sort(rules)
	}

	sort.Slice(rules, func(i, j int) bool {
		return ruleCompare(rules[i], rules[j])
	})

	for i := range rules {
		fmt.Printf("%q,\n", rules[i].Pattern)
	}
	return rules
}
