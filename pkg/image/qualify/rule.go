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

package qualify

import "strings"

// RuleError captures an invalid rule definition.
type RuleError struct {
	Definition     string
	Index          int
	InvalidDomain  string
	InvalidPattern string
	Message        string
}

type Rule struct {
	Domain string
	*pattern
}

// Error returns the parsing error.
func (p *RuleError) Error() string {
	return p.Message
}

func ParseRules(definitions []string) ([]Rule, error) {
	rules := make([]Rule, 0, len(definitions))

	for i, line := range definitions {
		words := strings.Fields(line)

		if len(words) == 0 {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(words[0]), "#") {
			continue
		}

		if len(words) != 2 {
			return nil, &RuleError{
				Definition: strings.TrimSpace(line),
				Index:      i + 1,
				Message:    "expected fields: pattern domain",
			}
		}

		pattern, err := parsePattern(words[0])
		if err != nil {
			return nil, &RuleError{
				Definition:     strings.TrimSpace(line),
				Index:          i + 1,
				InvalidPattern: words[0],
				Message:        err.Error(),
			}
		}

		if err := validateDomain(words[1]); err != nil {
			return nil, &RuleError{
				Definition:    strings.TrimSpace(line),
				Index:         i + 1,
				InvalidDomain: words[1],
				Message:       err.Error(),
			}
		}

		rules = append(rules, Rule{
			Domain:  words[1],
			pattern: pattern,
		})
	}

	return rules, nil
}
