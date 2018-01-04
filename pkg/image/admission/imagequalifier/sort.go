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
	"strings"
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
	sort.Stable(ms)
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

func orderRules(rules []Rule) []Rule {
	wild := make([]Rule, 0, len(rules))
	explicit := make([]Rule, 0, len(rules))

	for _, rule := range rules {
		if strings.Contains(rule.Pattern, "*") {
			wild = append(wild, rule)
		} else {
			explicit = append(explicit, rule)
		}
	}

	sort.Slice(rules, func(i int, j int) bool {
		x := strings.Count(rules[i].Pattern, "/")
		y := strings.Count(rules[j].Pattern, "/")
		if x != y {
			return x < y
		}
		return rules[i].Pattern < rules[j].Pattern
	})

	// sort.Slice(wild, func(i int, j int) bool {
	// 	x := strings.Count(wild[i].Pattern, "/")
	// 	y := strings.Count(wild[j].Pattern, "/")

	// 	if x != y {
	// 		return x > y
	// 	}

	// 	p := strings.Count(wild[i].Pattern, "*")
	// 	q := strings.Count(wild[j].Pattern, "*")

	// 	if p == q {
	// 		return wild[i].Pattern > wild[j].Pattern
	// 	}

	// 	return p > q
	// })

	// return append(explicit, wild...)
	return rules
}

func ruleCompare(a, b Rule) bool {
	if a.parts.digest < b.parts.digest {
		return true
	} else if b.parts.digest < a.parts.digest {
		return false
	}

	if a.parts.tag < b.parts.tag {
		return true
	} else if b.parts.tag < a.parts.tag {
		return false
	}

	if a.parts.image < b.parts.image {
		return true
	} else if b.parts.image < a.parts.image {
		return false
	}

	if a.parts.library < b.parts.library {
		return true
	} else if b.parts.library < a.parts.library {
		return false
	}

	if a.parts.domain < b.parts.domain {
		return true
	} else if b.parts.domain < a.parts.domain {
		return false
	}

	fmt.Printf("AAA: %#v\n", a.parts)
	fmt.Printf("bbb: %#v\n\n", b.parts)

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

	wild := make([]Rule, 0, len(rules))
	explicit := make([]Rule, 0, len(rules))

	for _, rule := range rules {
		if strings.Contains(rule.Pattern, "*") {
			wild = append(wild, rule)
		} else {
			explicit = append(explicit, rule)
		}
	}

	depth := func(x, y *pattern) bool {
		nx, ny := 0, 0
		if x.domain != "" {
			nx += 1
		}
		if y.domain != "" {
			ny += 1
		}
		return nx+strings.Count(x.path, "/") < ny+strings.Count(y.path, "/")
	}

	digest := func(x, y *pattern) bool {
		return string(x.digest) < string(y.digest)
	}

	tag := func(x, y *pattern) bool {
		return x.tag < y.tag
	}

	tagReverse := func(x, y *pattern) bool {
		return x.tag > y.tag
	}

	// path := func(x, y *pattern) bool {
	// 	return x.path < y.path
	// }

	image := func(x, y *pattern) bool {
		return x.image < y.image
	}

	library := func(x, y *pattern) bool {
		return x.library < y.library
	}

	domain := func(x, y *pattern) bool {
		return x.domain < y.domain
	}

	pattern := func(x, y *pattern) bool {
		return x.Rule.Pattern < y.Rule.Pattern
	}

	digest = digest
	tag = tag
	image = image
	library = library
	depth = depth
	domain = domain
	pattern = pattern
	tagReverse = tagReverse
	// OrderedBy(wtf).Sort(rules)
	// sort.Slice(rules, func(i, j int) bool {
	// 	return ruleCompare(rules[i], rules[j])
	// })

	OrderedBy(digest, tag, image, library, domain).Sort(explicit)
	OrderedBy(pattern).Sort(explicit)

	// sort.SliceStable(explicit, func(i, j int) bool {
	// 	return ruleCompare(explicit[i], explicit[j])
	// })

	sort.SliceStable(wild, func(i, j int) bool {
		return ruleCompare(wild[i], wild[j])
	})

	// OrderedBy(pattern).Sort(explicit)
	OrderedBy(pattern).Sort(wild)

	for i := range explicit {
		fmt.Printf("EXPLICIT %-03v - %s\n", i, explicit[i].Pattern)
	}

	for i := range wild {
		fmt.Printf("WILD     %-03v - %s\n", i, wild[i].Pattern)
	}

	for i, j := len(explicit)-1, 0; i >= 0; i, j = i-1, j+1 {
		rules[i] = explicit[j]
	}

	for i, j := len(wild)-1, 0; i >= 0; i, j = i-1, j+1 {
		rules[i+len(explicit)] = wild[j]
	}

	for i := range rules {
		fmt.Printf("%-03v - %s\n", i, rules[i].Pattern)
	}
	return rules

	// OrderedBy(pattern, digest).Sort(explicit)
	// OrderedBy(digest, tag, image, library, domain).Sort(wild)

	// for i := range explicit {
	// 	fmt.Println("wild   ", explicit[i])
	// }

	// for i := range wild {
	// 	fmt.Println("explict", wild[i])
	// }

	// return append(wild, explicit...)

	// sort.Slice(rules, func(i int, j int) bool {
	// 	return ruleCompare(rules[i], rules[j])
	// })
	// // OrderedBy(tag, image, library, domain).Sort(rules)
	// return rules
}
