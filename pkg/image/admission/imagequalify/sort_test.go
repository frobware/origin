package imagequalify_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	configapilatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/image/admission/imagequalify"
	"github.com/openshift/origin/pkg/image/admission/imagequalify/api"
)

func patterns(rules []api.ImageQualifyRule) []string {
	names := make([]string, len(rules))

	for i := range rules {
		names[i] = rules[i].Pattern
	}

	return names
}

func normaliseInput(input string) string {
	var bb bytes.Buffer

	for _, line := range strings.Split(input, "\n") {
		for _, word := range strings.Fields(line) {
			bb.WriteString(word)
			bb.WriteString(" ")
		}
	}

	return bb.String()
}

func patterns2config(input string) (*api.ImageQualifyConfig, error) {
	rules := []api.ImageQualifyRule{}

	for i, word := range strings.Fields(normaliseInput(input)) {
		rules = append(rules, api.ImageQualifyRule{
			Pattern: word,
			Domain:  fmt.Sprintf("domain%v.com", i),
		})
	}

	serializedConfig, serializationErr := configapilatest.WriteYAML(&api.ImageQualifyConfig{
		Rules: rules,
	})

	if serializationErr != nil {
		return nil, serializationErr
	}

	return imagequalify.ReadConfig(bytes.NewReader(serializedConfig))
}

func TestSort(t *testing.T) {
	var testcases = []struct {
		description string
		input       string
		expected    string
	}{{
		description: "default order is collating sequence",
		input:       "a b c",
		expected:    "c b a",
	}, {
		description: "wildcards have less priority",
		input:       "busybox:* busybox busybox:v1*",
		expected:    "busybox:v1* busybox:* busybox",
	}, {
		description: "wildcard default order is collating sequence",
		input:       "*/*/* * */*/*:latest */*/*@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff */*",
		expected:    "*/*/*@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff */*/*:latest */*/* */* *",
	}, {
		description: "default order is collating sequence, even for library components",
		input:       "foo/emacs foo/vim */* * emacs vim foo/*",
		expected:    "vim foo/vim foo/emacs foo/* emacs */* *",
	}, {
		description: "wild cards sort lower",
		input:       " */*/* */* a*b * abc",
		expected:    "abc a*b */*/* */* *",
	}, {
		description: "tags sort lower",
		input:       "abc * abc:latest abc:1.0 */*",
		expected:    "abc:latest abc:1.0 abc */* *",
	}, {
		description: "digests sort higher",
		input:       "abc */* abc@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff *",
		expected:    "abc@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff abc */* *",
	}, {
		description: "wildcard library references sort lower",
		input:       "foo/emacs */* * */vim emacs vim",
		expected:    "vim foo/emacs emacs */vim */* *",
	}, {
		description: "wildcard tags sort lower",
		input:       "* foo/emacs:* */vim */* foo/emacs emacs vim",
		expected:    "vim foo/emacs:* foo/emacs emacs */vim */* *",
	}, {
		description: "wildcard libraries",
		input:       "*me *you * */* */* */*/*",
		expected:    "*you *me */*/* */* */* *",
	}, {
		description: "library references",
		input: `
repo/busybox:latest
repo/busybox:1
repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
repo/busybox:*
repo/busybox
repo/busy
qwerty/busybox
*/*busy
repo/*
repo/busy*
busybox`,
		expected: `
repo/busybox@sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
repo/busybox:latest
repo/busybox:1
repo/busybox:*
repo/busybox
repo/busy*
repo/busy
repo/*
qwerty/busybox
busybox
*/*busy`,
	}}

	for i, tc := range testcases {
		config, err := patterns2config(tc.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := strings.Fields(normaliseInput(tc.expected))
		if !reflect.DeepEqual(expected, patterns(config.Rules)) {
			t.Errorf("test #%v: %s: expected %v, got %v", i, tc.description, expected, patterns(config.Rules))
		}
	}
}
