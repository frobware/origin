package imagequalify

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/golang/glog"
	configlatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/image/admission/imagequalify/api"
	"github.com/openshift/origin/pkg/image/admission/imagequalify/api/validation"
)

func filter(rules []api.ImageQualifyRule, test func(rule *api.ImageQualifyRule) bool) []api.ImageQualifyRule {
	filtered := make([]api.ImageQualifyRule, 0, len(rules))

	for i := range rules {
		if test(&rules[i]) {
			filtered = append(filtered, rules[i])
		}
	}

	return filtered
}

func sortRulesByPatterns(rules []api.ImageQualifyRule) {
	digest := func(x, y *PatternParts) bool {
		return string(x.Digest) > string(y.Digest)
	}

	tag := func(x, y *PatternParts) bool {
		return x.Tag > y.Tag
	}

	path := func(x, y *PatternParts) bool {
		return x.Path > y.Path
	}

	depth := func(x, y *PatternParts) bool {
		return strings.Count(x.Path, "/") > strings.Count(y.Path, "/")
	}

	explicitRules := filter(rules, func(rule *api.ImageQualifyRule) bool {
		return !strings.Contains(rule.Pattern, "*")
	})

	wildcardRules := filter(rules, func(rule *api.ImageQualifyRule) bool {
		return strings.Contains(rule.Pattern, "*")
	})

	// for i := range explicitRules {
	// 	fmt.Println("E:", explicitRules[i])
	// }

	// for i := range wildcardRules {
	// 	fmt.Println("W:", wildcardRules[i])
	// }

	sort.Stable(ByPatternDepth(rules))
	sort.Stable(ByPatternPathAscending(rules))
	sort.Stable(ByPatternTagAscending(rules))
	sort.Stable(ByPatternDigestAscending(rules))

	orderBy(depth, path, tag, digest).Sort(wildcardRules)
	// orderBy(tag).Stable(wildcardRules)
	// orderBy(digest).Stable(wildcardRules)
	// orderBy(depth).Stable(wildcardRules)

	orderBy(depth, path, tag, digest).Sort(explicitRules)
	// orderBy(digest).Stable(explicitRules)
	// orderBy(tag).Stable(explicitRules)
	// orderBy(depth).Stable(explicitRules)

	for i := range explicitRules {
		rules[i] = explicitRules[i]
	}

	for i := range wildcardRules {
		rules[i+len(explicitRules)] = wildcardRules[i]
	}
}

func readConfig(rdr io.Reader) (*api.ImageQualifyConfig, error) {
	obj, err := configlatest.ReadYAML(rdr)
	if err != nil {
		glog.V(5).Infof("%s error reading config: %v", api.PluginName, err)
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	config, ok := obj.(*api.ImageQualifyConfig)
	if !ok {
		return nil, fmt.Errorf("unexpected config object: %#v", obj)
	}
	glog.V(5).Infof("%s config is: %#v", api.PluginName, config)
	if errs := validation.Validate(config); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}
	if len(config.Rules) > 0 {
		sortRulesByPatterns(config.Rules)
	}
	return config, nil
}
