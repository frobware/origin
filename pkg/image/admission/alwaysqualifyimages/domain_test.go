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

package alwaysqualifyimages

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	for i, domain := range []string{
		"test.io",
		"localhost",
		"localhost:5000",
		"a.b.c.d.e.f",
		"a.b.c.d.e.f:5000",
	} {
		if err := ValidateDomain(domain); err != nil {
			t.Errorf("test #%d: unexpected error for %q, got %v", i, domain, err)
		}
	}
}

func TestValidateNameErrors(t *testing.T) {
	for i, test := range []struct {
		description string
		input       string
	}{{
		description: "empty input",
		input:       "",
	}, {
		description: "bad characters in domain name",
		input:       "!invalidname!",
	}, {
		description: "no '.' or :<PORT> and not 'localhost'",
		input:       "domain",
	}, {
		description: "a valid name but too long",
		input:       strings.Repeat("x", 255) + ".io",
	}} {
		if err := ValidateDomain(test.input); err == nil {
			t.Errorf("test #%v: expected an error for %q", i, test.description)
		}
	}
}
