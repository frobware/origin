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
		description: "default order is ascending order",
		input:       "a b c",
		expected:    "c b a",
	}, {
		description: "wildcards sort last",
		input:       "a* a",
		expected:    "a a*",
	}, {
		description: "explicit patterns, followed by wildcard patterns",
		input:       "busybox:* busybox:v1.2.3* a busybox:v1.2* b busybox busybox:v1* c nginx",
		expected:    "nginx c busybox b a busybox:v1.2.3* busybox:v1.2* busybox:v1* busybox:*",
	}, {
		description: "wildcards only",
		input:       "* */* */*/*",
		expected:    "*/*/* */* *",
	}, {
		description: "explicit followed by wildcards",
		input:       "* */* */*/* a/a b/a c/a c b a",
		expected:    "c/a c b/a b a/a a */*/* */* *",
	}, {
		description: "patterns with tags sort in ascending order",
		input:       "abc:* abc * a b c abc:latest b*:* abc:1.0 */*",
		expected:    "c b abc:latest abc:1.0 abc a b*:* abc:* */* *",
	}, {
		description: "patterns with digest sort in ascending order",
		input:       "abc */* * abc@sha256:ee */abc@sha256:ff */@*",
		expected:    "abc@sha256:ee abc */abc@sha256:ff */@* */* *",
	}, {
		description: "wildcard repositories sort first",
		input:       "*me *you * */* */* */*/*",
		expected:    "*you *me */*/* */* */* *",
	}}

	for i, tc := range testcases {
		config, err := patterns2config(tc.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := strings.Fields(normaliseInput(tc.expected))
		if !reflect.DeepEqual(expected, patterns(config.Rules)) {
			t.Errorf("test #%v: %s: expected %v, got %v", i, tc.description, tc.expected, patterns(config.Rules))
		}
	}
}
