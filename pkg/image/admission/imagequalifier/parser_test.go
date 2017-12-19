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
	"testing"

	"github.com/openshift/origin/pkg/image/admission/imagequalifier"
)

func TestParseNilInput(t *testing.T) {
	if _, err := imagequalifier.ParseInput("", nil); err == nil {
		t.Fatalf("expected an error")
	}
}

func TestParseOpenDirectoryErrors(t *testing.T) {
	_, err := imagequalifier.ParseInput("testdata", nil)
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "read testdata: is a directory"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err)
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := imagequalifier.ParseInput("testdata/does-not-exist", nil)
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "open testdata/does-not-exist: no such file or directory"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err)
	}
}

func TestParseEmptyFilename(t *testing.T) {
	rules, err := imagequalifier.ParseInput("testdata/emptyfile", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules")
	}
}

func TestParseEmptyInput(t *testing.T) {
	rules, err := imagequalifier.ParseInput("", "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules")
	}
}

func TestParseInvalidInputErrors(t *testing.T) {
	if _, err := imagequalifier.ParseInput("", map[string]bool{}); err == nil {
		t.Fatalf("expected an error")
	}
}

func TestParseLineFormatErrors(t *testing.T) {
	content := `
# Too many fields
a b c`
	_, err := imagequalifier.ParseInput("", content)
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "invalid field count; expected <pattern> <domain>"

	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestParseInvalidPattern(t *testing.T) {
	content := `
[]a] foo.io`
	_, err := imagequalifier.ParseInput("", content)
	if err == nil {
		t.Fatalf("expected an error")
	}

	ruleErr, ok := err.(*imagequalifier.RuleError)
	if !ok {
		t.Fatalf("expected a RuleError; got %T", err)
	}

	if ruleErr.Filename != "" {
		t.Errorf(`filename should be ""`)
	}

	if ruleErr.Line != content[1:] {
		t.Errorf("expected input=%q, got %q", content[1:], ruleErr.Line)
	}

	if ruleErr.LineNum != 2 {
		t.Errorf("expected error on line 2, got %v", ruleErr.LineNum)
	}

	expected := "syntax error in pattern"

	if expected != ruleErr.Error() {
		t.Errorf("expected error %q, got %q", expected, ruleErr)
	}
}

func TestParseInvalidDomain(t *testing.T) {
	badrule := "busybox !foo.io!"
	content := `
# A comment line, followed by an empty blank line.

`
	content += badrule

	_, err := imagequalifier.ParseInput("", content)
	if err == nil {
		t.Fatalf("expected an error")
	}

	ruleErr, ok := err.(*imagequalifier.RuleError)
	if !ok {
		t.Fatalf("expected a RuleError; got %T", err)
	}

	if ruleErr.Filename != "" {
		t.Errorf(`Filename should be ""`)
	}

	if ruleErr.Line != badrule {
		t.Errorf("expected input=%q, got %q", badrule, ruleErr.Line)
	}

	if ruleErr.LineNum != 4 {
		t.Errorf("expected error on line 4, got %v", ruleErr.LineNum)
	}

	expected := "invalid reference format"

	if expected != ruleErr.Error() {
		t.Errorf("expected error %q, got %q", expected, ruleErr)
	}
}

func TestParseKnownGoodRules(t *testing.T) {
	rules, err := imagequalifier.ParseInput("testdata/parser-rules", nil)
	if err != nil {
		t.Fatalf("unexpected error; got %s", err)
	}

	if len(rules) != 6 {
		t.Errorf("expected 6 rules, got %v", len(rules))
	}
}
