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

import (
	"fmt"
	"strings"
)

// RuleError captures an invalid rule definition.
type RuleError struct {
	Definition     string
	LineNumber     int
	InvalidDomain  string // split these into discrete errors
	InvalidPattern string // split these into discrete errors
	Message        string
}

type Rule struct {
	Domain string
	*pattern
}

// Error returns the parsing error.
func (p RuleError) Error() string {
	return fmt.Sprintf("line %v: invalid definition: %q: %s", p.LineNumber, p.Definition, p.Message)
}

func ParseRules(input string) ([]Rule, error) {
	lines := strings.Split(input, "\n")
	rules := make([]Rule, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			// Skip blank lines and comments
			continue
		}

		words := strings.Fields(line)
		if len(words) != 2 {
			return nil, &RuleError{
				Definition: strings.TrimSpace(line),
				LineNumber: i + 1,
				Message:    "expected fields: pattern domain",
			}
		}

		pattern, err := parsePattern(words[0])
		if err != nil {
			return nil, &RuleError{
				Definition:     line,
				LineNumber:     i + 1,
				InvalidPattern: words[0],
				Message:        err.Error(),
			}
		}

		if err := validateDomain(words[1]); err != nil {
			return nil, &RuleError{
				Definition:    line,
				LineNumber:    i + 1,
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
