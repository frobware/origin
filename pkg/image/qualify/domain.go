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

package qualify

import (
	"errors"
)

const imageRefWithoutDomain = "foo/bar:latest"

// validateDomain validates that input (e.g., "myregistry.io") can be
// used as the domain component in a docker image reference. Returns
// an error if domain would be invalid.
func xxxvalidateDomain(domain string) error {
	matchedDomain, remainder, err := SplitImageName(domain + "/" + imageRefWithoutDomain)
	if err != nil {
		return err
	}
	if domain != matchedDomain && remainder != imageRefWithoutDomain {
		return errors.New("invalid domain")
	}
	return nil
}

func validateDomain(input string) error {
	matches := DomainRegexp.FindStringSubmatch(input)
	if matches == nil {
		return errors.New("invalid domain")
	}
	domain, _, err := SplitImageName(input + "/" + imageRefWithoutDomain)
	if err != nil {
		return err
	}
	if domain != input {
		return errors.New("invalid domain 2")
	}
	return nil
}
