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
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

// RuleError captures an invalid rule definition.
type RuleError struct {
	Filename string // maybe "" if read from []byte or string
	Line     string
	LineNum  int
	Message  string
}

// Error returns the parsing error.
func (p *RuleError) Error() string {
	return p.Message
}

// If src != nil, consume converts src to a []byte if possible,
// otherwise it returns an error. If src == nil, consume returns the
// result of reading the complete contents of filename, or an error if
// consuming filename fails.
func consume(filename string, src interface{}) ([]byte, error) {
	if src == nil {
		return ioutil.ReadFile(filename)
	}
	switch s := src.(type) {
	case string:
		return []byte(s), nil
	}
	return nil, errors.New("invalid source type")
}

func readLines(input io.Reader) []string {
	lines := []string{}
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func parseInput(filename string, src interface{}) ([]Rule, error) {
	content, err := consume(filename, src)
	if err != nil {
		return nil, err
	}

	lines := readLines(bytes.NewReader(content))
	rules := make([]Rule, 0, len(lines))

	for i, line := range lines {
		words := strings.Fields(line)

		if len(words) == 0 {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(words[0]), "#") {
			continue
		}

		if len(words) != 2 {
			return nil, &RuleError{
				Line:     line,
				LineNum:  i + 1,
				Filename: filename,
				Message:  "invalid field count; expected <pattern> <domain>",
			}
		}

		if _, err := path.Match(words[0], "doesnotmatter"); err != nil {
			return nil, &RuleError{
				Line:     line,
				LineNum:  i + 1,
				Filename: filename,
				Message:  err.Error(),
			}
		}

		if err := validateDomain(words[1]); err != nil {
			return nil, &RuleError{
				Line:     line,
				LineNum:  i + 1,
				Filename: filename,
				Message:  err.Error(),
			}
		}

		pattern, err := parsePattern(words[0])
		if err != nil {
			return nil, &RuleError{
				Line:     line,
				LineNum:  i + 1,
				Filename: filename,
				Message:  err.Error(),
			}
		}

		rules = append(rules, Rule{
			Domain:  words[1],
			pattern: pattern,
		})
	}

	return rules, nil
}

func ParseRules(filename string) ([]Rule, error) {
	return parseInput(filename, nil)
}
