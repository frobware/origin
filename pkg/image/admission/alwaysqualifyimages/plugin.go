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
	"io"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/image/admission/alwaysqualifyimages/apis/alwaysqualifyimages/validation"
	"k8s.io/apiserver/pkg/admission"
)

// Register registers a plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register("AlwaysQualifyImages", func(config io.Reader) (admission.Interface, error) {
		configuration, err := loadConfiguration(config)
		if err != nil {
			return nil, err
		}
		// validate the configuration (if any)
		if configuration != nil {
			if errs := validation.ValidateConfiguration(configuration); len(errs) != 0 {
				return nil, errs.ToAggregate()
			}
		}
		if err := ValidateDomain(configuration.Domain); err != nil {
			return nil, err
		}
		glog.V(2).Infof("AlwaysQualifyImages %+v", configuration)
		return NewAlwaysQualifyImages(configuration.Domain), nil
	})
}
