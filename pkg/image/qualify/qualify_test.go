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
	"regexp"
	"runtime"
	"testing"

	"github.com/openshift/origin/pkg/image/qualify"
)

type testcase struct {
	image  string
	domain string
}

// Match Test<XXX> func name in a package specifier.
//
// For example, it will match TestFoo in:
//   github.io/xxx/foo_test.TestFoo
var testNameRegexp = regexp.MustCompile(`\.(Test[\p{L}_\p{N}]+)$`)

func testName() string {
	pc := make([]uintptr, 32)
	n := runtime.Callers(0, pc)

	for i := 0; i < n; i++ {
		fn := runtime.FuncForPC(pc[i])
		matches := testNameRegexp.FindStringSubmatch(fn.Name())
		if matches == nil {
			continue
		}
		return matches[1]
	}

	panic("test name could not be discovered; try increasing stack depth?")
}

func testQualify(t *testing.T, input string, tests []testcase) {
	rules, err := qualify.ParseInput("", input)

	if err != nil {
		t.Fatalf("unexpected error; got %s", err)
	}

	for i, tc := range tests {
		//t.Logf("%s: test #%v: %q", testName(), i, tc.image)
		domain, qualifiedImage := qualify.Qualify(tc.image, rules)

		if domain != tc.domain {
			t.Errorf("%s: test #%v: expected domain %q, got %q", testName(), i, tc.domain, domain)
		}

		if domain == "" {
			continue
		}

		// This is a sanity check to assert that the resultant
		// image is valid and the constituent parts match the
		// test case inputs.

		domain, remainder, err := qualify.SplitImageName(qualifiedImage)
		if err != nil {
			t.Fatalf("unexpected error; got %s", err)
		}
		if domain != tc.domain {
			t.Errorf("%s: test #%v: expected domain %q, got %q", testName(), i, tc.domain, domain)
		}
		if remainder != tc.image {
			t.Errorf("%s: test #%v: expected image %q, got %q", testName(), i, tc.image, remainder)
		}
	}
}

func TestQualifyInvalidInput(t *testing.T) {
	_, err := qualify.ParseInput("", map[string]bool{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestQualifyNoRules(t *testing.T) {
	tests := []testcase{{
		image:  "busybox",
		domain: "",
	}, {
		image:  "repo/busybox",
		domain: "",
	}}

	testQualify(t, "", tests)
}

func TestQualifyImageNoMatch(t *testing.T) {
	rules := `
    busybox      production.io
    busybox:v1*  v1.io
    busybox:*    next.io`

	tests := []testcase{{
		image:  "nginx",
		domain: "",
	}, {
		image:  "nginx:latest",
		domain: "",
	}, {
		image:  "repo/nginx",
		domain: "",
	}, {
		image:  "repo/nginx:latest",
		domain: "",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyRepoAndImageAndTagsWithWildcard(t *testing.T) {
	rules := `
    repo/busybox            production.io
    repo/busybox:v1*        v1.io
    repo/busybox:v[7-9]*    v7.io
    repo/busybox:*          next.io`

	tests := []testcase{{
		image:  "busybox",
		domain: "",
	}, {
		image:  "busybox:latest",
		domain: "",
	}, {
		image:  "repo/busybox",
		domain: "production.io",
	}, {
		image:  "repo/busybox:v1.2.3",
		domain: "v1.io",
	}, {
		image:  "repo/busybox:v5",
		domain: "next.io",
	}, {
		image:  "repo/busybox:v7",
		domain: "v7.io",
	}, {
		image:  "repo/busybox:v9",
		domain: "v7.io",
	}, {
		image:  "repo/busybox:latest",
		domain: "next.io",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyNoRepoWithImageWildcard(t *testing.T) {
	rules := `* default.io`

	tests := []testcase{{
		image:  "nginx",
		domain: "default.io",
	}, {
		image:  "repo/nginx",
		domain: "",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyRepoAndImageWildcard(t *testing.T) {
	rules := `
    */* repo.io
    *   default.io`

	tests := []testcase{{
		image:  "nginx",
		domain: "default.io",
	}, {
		image:  "repo/nginx",
		domain: "repo.io",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyWildcards(t *testing.T) {
	rules := `
    */*:* first.io
    */*   second.io
    *     third.io`

	tests := []testcase{{
		image:  "busybox",
		domain: "third.io",
	}, {
		image:  "busybox:latest",
		domain: "third.io",
	}, {
		image:  "nginx",
		domain: "third.io",
	}, {
		image:  "repo/busybox:latest",
		domain: "first.io",
	}, {
		image:  "repo/busybox",
		domain: "second.io",
	}, {
		image:  "repo/nginx",
		domain: "second.io",
	}, {
		image:  "nginx",
		domain: "third.io",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyRepoWithWildcards(t *testing.T) {
	rules := `
    a*/*:* a-with-tag.io
    b*/*:* b-with-tag.io

    a*/*   a.io
    b*/*   b.io

    */*:*  first.io
    */*    second.io
    *      third.io`

	tests := []testcase{{
		image:  "abc/nginx",
		domain: "a.io",
	}, {
		image:  "bcd/nginx",
		domain: "b.io",
	}, {
		image:  "nginx",
		domain: "third.io",
	}, {
		image:  "repo/nginx",
		domain: "second.io",
	}, {
		image:  "repo/nginx:latest",
		domain: "first.io",
	}, {
		image:  "abc/nginx:1.0",
		domain: "a-with-tag.io",
	}, {
		image:  "bcd/nginx:1.0",
		domain: "b-with-tag.io",
	}}

	testQualify(t, rules, tests)
}

func TestQualifyTagsWithWildcards(t *testing.T) {
	rules := `
    a*/*:*v1* v1.io
    a*/*:*v2* v2.io
    a*/*:*v*  v3.io`

	tests := []testcase{{
		image:  "abc/nginx",
		domain: "",
	}, {
		image:  "bcd/nginx",
		domain: "",
	}, {
		image:  "abc/nginx:v1.0",
		domain: "v1.io",
	}, {
		image:  "abc/nginx:v2.0",
		domain: "v2.io",
	}, {
		image:  "abc/nginx:v0",
		domain: "v3.io",
	}, {
		image:  "abc/nginx:latest",
		domain: "",
	}}

	testQualify(t, rules, tests)
}
