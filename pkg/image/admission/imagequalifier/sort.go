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

func isWild(rule Rule) bool {
	return strings.Contains(rule.Pattern, "*")
}
func sortRules(rules []Rule) []Rule {
	wild := make([]Rule, 0, len(rules))
	explicit := make([]Rule, 0, len(rules))

	for _, rule := range rules {
		reference, err := parsePattern(rule.Pattern)

		fmt.Printf("%v\n", err)
		fmt.Printf("%#v\n", reference)

		if isWild(rule) {
			wild = append(wild, rule)
		} else {
			explicit = append(explicit, rule)
		}
	}

	sort.SliceStable(wild, func(i, j int) bool {
		return wild[i].Pattern > wild[j].Pattern
	})

	sort.SliceStable(explicit, func(i, j int) bool {
		return explicit[i].Pattern < explicit[j].Pattern
	})

	return append(explicit, wild...)
}
