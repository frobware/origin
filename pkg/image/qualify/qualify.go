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
	"fmt"
	"path"
)

// Qualify unqualifiedImage to include a domain component based on a
// set of pattern matching rules. If no rule matches then "" is
// returned for both domain and qualifiedImage. If a match is found,
// then domain contains the discrete domain and qualifiedImage returns
// "<domain>/<unqualifiedImage>".
//
// unqualifiedImage must be unqualified (i.e., it must have no domain
// component) and not be the empty string.
func Qualify(unqualifiedImage string, rules []Rule) (domain string, qualifiedImage string) {
	for i := range rules {
		if ok, _ := path.Match(rules[i].Pattern, unqualifiedImage); ok {
			return rules[i].Domain, fmt.Sprintf("%s/%s", rules[i].Domain, unqualifiedImage)
		}
	}
	return "", ""
}
