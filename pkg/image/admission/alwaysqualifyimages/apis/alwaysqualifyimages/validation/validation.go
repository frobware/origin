package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	alwaysqualifyimagesapi "github.com/openshift/origin/pkg/image/admission/alwaysqualifyimages/apis/alwaysqualifyimages"
)

// ValidateConfiguration validates the configuration.
func ValidateConfiguration(config *alwaysqualifyimagesapi.Configuration) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}
