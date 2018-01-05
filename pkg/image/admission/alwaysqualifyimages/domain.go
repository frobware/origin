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
	"errors"

	"k8s.io/kubernetes/pkg/util/parsers"
)

const sanityRepo = "foo/bar:latest"

// ValidateDomain validates that domain (e.g., "myregistry.io") can be
// used as the registry component for a docker image reference.
// Returns an error if domain would be invalid.
func ValidateDomain(domain string) error {
	registry, remainder, err := parsers.SplitImageName(domain + "/" + sanityRepo)
	if err != nil {
		return err
	}
	if registry != domain && remainder != sanityRepo {
		return errors.New("invalid domain")
	}
	return nil
}
