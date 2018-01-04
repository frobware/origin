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
	"fmt"
	"sort"
)

type lessFunc func(x, y *pattern) bool

type ruleSorter struct {
	rules []Rule
	less  []lessFunc
}

// Sort sorts the argument slice according to the less functions
// passed to OrderedBy.
func (ms *ruleSorter) Sort(rules []Rule) {
	ms.rules = rules
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *ruleSorter {
	return &ruleSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *ruleSorter) Len() int {
	return len(ms.rules)
}

// Swap is part of sort.Interface.
func (ms *ruleSorter) Swap(i, j int) {
	ms.rules[i], ms.rules[j] = ms.rules[j], ms.rules[i]
}

// Less is part of sort.Interface. It is implemented by looping along
// the less functions until it finds a comparison that is either Less
// or !Less. Note that it can call the less functions twice per call.
// We could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
func (ms *ruleSorter) Less(i, j int) bool {
	p, q := ms.rules[i], ms.rules[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p.parts, q.parts):
			// p < q, so we have a decision.
			return true
		case less(q.parts, p.parts):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}

	// All comparisons to here are "equal", so just return
	// whatever the final comparison reports.

	return ms.less[k](p.parts, q.parts)
}

// func unglobPattern(p *pattern) string {
// 	x := fmt.Sprintf("/%s/%s/%s:%s/%s",
// 		unglob(p.parts.domain),
// 		unglob(p.parts.library),
// 		unglob(p.parts.image),
// 		unglob(p.parts.tag),
// 		unglob(string(p.parts.digest)))
// 	return x
// }

// func globifyPattern(p *pattern) {
// 	if p.digest == "" {
// 		p.digest = "*"
// 	}
// 	if p.tag == "" {
// 		p.tag = "*"
// 	}
// 	if p.image == "" {
// 		p.image = "*"
// 	}
// 	if p.library == "" {
// 		p.library = "*"
// 	}
// 	if p.domain == "" {
// 		p.domain = "*"
// 	}
// }

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
	if globcmp(string(a.parts.digest), string(b.parts.digest)) {
		return true
	} else if globcmp(string(b.parts.digest), string(a.parts.digest)) {
		return false
	}

	if globcmp(a.parts.tag, b.parts.tag) {
		return true
	} else if globcmp(b.parts.tag, a.parts.tag) {
		return false
	}

	if globcmp(a.parts.image, b.parts.image) {
		return true
	} else if globcmp(b.parts.image, a.parts.image) {
		return false
	}

	if globcmp(a.parts.library, b.parts.library) {
		return true
	} else if globcmp(b.parts.library, a.parts.library) {
		return false
	}

	if globcmp(a.parts.domain, b.parts.domain) {
		return true
	} else if globcmp(b.parts.domain, a.parts.domain) {
		return false
	}

	return false
}

func sortRules(rules []Rule) []Rule {
	for i := range rules {
		p, err := parsePattern(rules[i].Pattern)
		if err != nil {
			panic(p)
		}
		rules[i].parts = p
		p.Rule = &rules[i]
	}

	digest := func(x, y *pattern) bool {
		return globcmp(string(x.digest), string(y.digest))
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

	OrderedBy(digest, tag, image, library, domain).Sort(rules)

	for i := range rules {
		fmt.Printf("%-03v - %s\n", i, rules[i].Pattern)
	}
	return rules
}
