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
	"testing"

	"github.com/openshift/origin/pkg/image/qualify"
)

func TestParseNilInput(t *testing.T) {
	if _, err := qualify.ParseDefinitions(""); err == nil {
		t.Fatalf("expected an error")
	}
}

func TestParseOpenDirectoryErrors(t *testing.T) {
	_, err := qualify.ParseDefinitions("testdata")
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "read testdata: is a directory"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err)
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := qualify.ParseDefinitions("testdata/does-not-exist")
	if err == nil {
		t.Fatalf("expected an error")
	}

	expected := "open testdata/does-not-exist: no such file or directory"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err)
	}
}

func TestParseEmptyFilename(t *testing.T) {
	rules, err := qualify.ParseDefinitions("testdata/emptyfile")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules")
	}
}

func TestParseEmptyInput(t *testing.T) {
	rules, err := qualify.ParseDefinitions("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules")
	}
}
