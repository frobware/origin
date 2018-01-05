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

package qualifier

import (
	"strings"

	"k8s.io/kubernetes/pkg/util/parsers"
)

// SplitImageName splits a docker image string into the domain
// component and path components. An empty string is returned if there
// is no domain component. This function will first validate that
// image is a valid reference, returning an error if it is not.
// Validation is done without normalising the image.
//
// Examples inputs and results for the domain component:
//
//   "busybox"                    -> domain is ""
//   "foo/busybox"                -> domain is ""
//   "localhost/foo/busybox"      -> domain is "localhost"
//   "localhost:5000/foo/busybox" -> domain is "localhost:5000"
//   "gcr.io/busybox"             -> domain is "gcr.io"
//   "gcr.io/foo/busybox"         -> domain is "gcr.io"
//   "docker.io/busybox"          -> domain is "docker.io"
//   "docker.io/library/busybox"  -> domain is "docker.io"
func SplitImageName(image string) (string, string, error) {
	if _, _, _, err := parsers.ParseImageName(image); err != nil {
		return "", "", err
	}
	i := strings.IndexRune(image, '/')
	if i == -1 || (!strings.ContainsAny(image[:i], ".:") && image[:i] != "localhost") {
		return "", image, nil
	} else {
		return image[:i], image[i+1:], nil
	}
}
