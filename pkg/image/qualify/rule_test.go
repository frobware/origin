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

package qualify_test

import (
	"strings"
	"testing"

	"github.com/openshift/origin/pkg/image/qualify"
)

func TestRuleLineInvalidFieldCount(t *testing.T) {
	_, err := qualify.ParseRules(strings.Split("a b c", "\n"))
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "expected fields: pattern domain"

	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestRuleInvalidPattern(t *testing.T) {
	invalidPattern := "[]a]"

	_, err := qualify.ParseRules(strings.Split(invalidPattern+" foo.io", "\n"))
	if err == nil {
		t.Fatalf("expected an error")
	}

	ruleErr, ok := err.(*qualify.RuleError)
	if !ok {
		t.Fatalf("expected a RuleError; got %T", err)
	}

	if ruleErr.InvalidPattern != invalidPattern {
		t.Errorf("expected pattern=%q, got=%q", "[[a]", ruleErr.InvalidPattern)
	}

	if ruleErr.Index != 1 {
		t.Errorf("expected error on line 1, got %v", ruleErr.Index)
	}
}

func TestRuleInvalidDomain(t *testing.T) {
	invalidDomain := "!foo.io"

	_, err := qualify.ParseRules(strings.Split("busybox "+invalidDomain, "\n"))
	if err == nil {
		t.Fatalf("expected an error")
	}

	ruleErr, ok := err.(*qualify.RuleError)
	if !ok {
		t.Fatalf("expected a RuleError; got %T", err)
	}

	if ruleErr.InvalidDomain != invalidDomain {
		t.Errorf("expected %q, got %q", invalidDomain, ruleErr.InvalidDomain)
	}

	if ruleErr.Index != 1 {
		t.Errorf("expected error on line 1, got %v", ruleErr.Index)
	}
}
