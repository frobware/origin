package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/origin/pkg/image/admission/imagequalify/api"
)

func Validate(config *api.ImageQualifyConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if config == nil {
		return allErrs
	}
	for i, rule := range config.Rules {
		if rule.Pattern == "" {
			allErrs = append(allErrs, field.Required(field.NewPath(api.PluginName, "rules").Index(i).Child("pattern"), ""))
		}
		if rule.Domain == "" {
			allErrs = append(allErrs, field.Required(field.NewPath(api.PluginName, "rules").Index(i).Child("domain"), ""))
		}
		if rule.Domain != "" {
			if err := validateDomain(rule.Domain); err != nil {
				allErrs = append(allErrs, field.Invalid(field.NewPath(api.PluginName, "rules").Index(i).Child("domain"), rule.Domain, err.Error()))
			}
		}
	}
	return allErrs
}
