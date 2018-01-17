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

	if len(config.Rules) == 0 {
		return config, nil
	}

	explicitRules := filter(config.Rules, func(rule *api.ImageQualifyRule) bool {
		return !strings.Contains(rule.Pattern, "*")
	})

	wildcardRules := filter(config.Rules, func(rule *api.ImageQualifyRule) bool {
		return strings.Contains(rule.Pattern, "*")
	})

	sort.Sort(ByPatternAscending(explicitRules))
	sort.Sort(ByPatternAscending(wildcardRules))
	config.Rules = append(explicitRules, wildcardRules...)

	return config, nil
}
