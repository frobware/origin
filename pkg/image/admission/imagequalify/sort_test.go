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

func patternsFromRules(rules []api.ImageQualifyRule) string {
	var bb bytes.Buffer

	for i := range rules {
		bb.WriteString(rules[i].Pattern)
		bb.WriteString(" ")
	}

	return strings.TrimSpace(bb.String())
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

func parseTestSortPatterns(input string) (*api.ImageQualifyConfig, error) {
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

func filter(rules []api.ImageQualifyRule, test func(rule *api.ImageQualifyRule) bool) []api.ImageQualifyRule {
	filtered := make([]api.ImageQualifyRule, 0, len(rules))

	for i := range rules {
		if test(&rules[i]) {
			filtered = append(filtered, rules[i])
		}
	}

	return filtered
}

func TestSort(t *testing.T) {
	var testcases = []struct {
		description string
		input       string
		expected    string
	}{{
		description: "default order is lexicographical (ascending)",
		input:       "a b c",
		expected:    "c b a",
	}, {
		description: "longer patterns sort first",
		input:       "a b/c c/d b/c/d/e b/c/d/f b/c/d b/c/d/f/f",
		expected:    "b/c/d/f/f b/c/d/f b/c/d/e b/c/d c/d b/c a",
	}, {
		description: "longer patterns sort first",
		input:       "* */c c/*",
		expected:    "c/* */c *",
	}, {
		description: "wildcards sort last",
		input:       "a* *m *ma *a y m a a*m*",
		expected:    "y m a a*m* a* *ma *m *a",
	}, {
		description: "longer paths sort before shorter",
		input:       "a a/b a/b/c",
		expected:    "a/b/c a/b a",
	}, {
		description: "tags with longer paths sort before shorter",
		input:       "a a/b a/b/c x:latest x/y:latest x/y/z:latest",
		expected:    "x/y/z:latest x/y:latest x:latest a/b/c a/b a",
	}, {
		description: "explicit patterns, followed by wildcard patterns",
		input:       "busybox:* busybox:v1.2.3* a busybox:v1.2* b busybox busybox:v1* c nginx",
		expected:    "nginx c busybox b a busybox:v1.2.3* busybox:v1.2* busybox:v1* busybox:*",
	}, {
		description: "wildcards only",
		input:       "* */* */*/*",
		expected:    "*/*/* */* *",
	}, {
		description: "explicit patterns come before all wildcard patterns",
		input:       "* */* */*/* *a/a b/*a *c*/*a* c b a a/a b/a c/a",
		expected:    "c/a b/a a/a c b a */*/* b/*a *c*/*a* *a/a */* *",
	}, {
		description: "patterns with tags sort in ascending order",
		input:       "abc:* abc * a b c abc:latest b*:* abc:1.0 */*",
		expected:    "abc:latest abc:1.0 c b abc a b*:* abc:* */* *",
	}, {
		description: "patterns with digests sort in ascending order",
		input:       "abc */* * abc@sha256:ee */abc@sha256:ff */@*",
		expected:    "abc@sha256:ee abc */abc@sha256:ff */@* */* *",
	}, {
		description: "wildcard repositories sort first",
		input:       "y *m m *my",
		expected:    "y m *my *m",
	}}

	for i, tc := range testcases {
		rules := make([]api.ImageQualifyRule, 0)

		for j, word := range strings.Fields(normaliseInput(tc.input)) {
			rules = append(rules, api.ImageQualifyRule{
				Pattern: word,
				Domain:  fmt.Sprintf("domain%v.com", j),
			})
		}

		explicitRules := filter(rules, func(rule *api.ImageQualifyRule) bool {
			return !strings.Contains(rule.Pattern, "*")
		})

		wildcardRules := filter(rules, func(rule *api.ImageQualifyRule) bool {
			return strings.Contains(rule.Pattern, "*")
		})

		fmt.Printf("-- test %#v: %s\n", i, tc.description)
		fmt.Println("drop table if exists explicit;\ncreate table explicit(tc text, pattern text, path text, tag text, digest text, depth number);\n")
		fmt.Println("drop table if exists wildcard;\ncreate table wildcard(tc text, pattern text, path text, tag text, digest text, depth number);\n")

		for j := range explicitRules {
			parts := imagequalify.DestructurePattern(explicitRules[j].Pattern)
			fmt.Printf("insert into explicit values(%q, %q, %q, %q, %q, %v);\n", tc.input, parts.Pattern, parts.Path, parts.Tag, parts.Digest, parts.Depth)
		}
		for j := range wildcardRules {
			parts := imagequalify.DestructurePattern(wildcardRules[j].Pattern)
			fmt.Printf("insert into wildcard values(%q, %q, %q, %q, %q, %v);\n", tc.input, parts.Pattern, parts.Path, parts.Tag, parts.Digest, parts.Depth)
		}
		fmt.Printf("select group_concat(explicit.pattern, %q) from explicit order by explicit.digest DESC, explicit.tag DESC, explicit.depth DESC, explicit.path DESC;\n\n", " ")
		fmt.Printf("select group_concat(wildcard.pattern, %q) from wildcard order by wildcard.digest DESC, wildcard.tag DESC, wildcard.depth DESC, wildcard.path DESC;\n\n", " ")
	}

	for i, tc := range testcases {
		config, err := parseTestSortPatterns(tc.input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		actualPatterns := patternsFromRules(config.Rules)

		if !reflect.DeepEqual(tc.expected, actualPatterns) {
			t.Errorf("test #%v: %s: expected [%s], got [%s]", i, tc.description, tc.expected, actualPatterns)
		}
	}

}
