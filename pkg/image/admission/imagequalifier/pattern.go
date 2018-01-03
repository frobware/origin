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

package imagequalifier

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"
)

type pattern struct {
	domain string
	path   string
	tag    string
	digest digest.Digest
}

func splitDomain(image string) (string, string) {
	i := strings.IndexRune(image, '/')
	if i == -1 || (!strings.ContainsAny(image[:i], ".:") && image[:i] != "localhost") {
		return "", image
	}
	return image[:i], image[i+1:]
}

func parsePattern(s string) (*pattern, error) {
	matches := ReferenceRegexp.FindStringSubmatch(s)
	fmt.Printf("MATCHES: %#v\n", matches)
	if matches == nil {
		if s == "" {
			return nil, reference.ErrNameEmpty
		}
		if ReferenceRegexp.FindStringSubmatch(strings.ToLower(s)) != nil {
			return nil, reference.ErrNameContainsUppercase
		}
		return nil, reference.ErrReferenceInvalidFormat
	}

	if len(matches[1]) > 2500 {
		return nil, errors.New("ErrNameTooLong")
	}

	var ref pattern

	// nameMatch := anchoredNameRegexp.FindStringSubmatch(matches[1])
	// if nameMatch != nil && len(nameMatch) == 3 {
	// 	ref.domain = nameMatch[1]
	// 	ref.path = nameMatch[2]
	// } else {
	// 	image := matches[1]
	// 	i := strings.IndexRune(image, '/')
	// 	if i == -1 || (!strings.ContainsAny(image[:i], ".:") && image[:i] != "localhost") {
	// 		ref.domain, ref.path = "", image
	// 	} else {
	// 		ref.domain, ref.path = image[:i], image[i+1:]
	// 	}
	// }

	ref.domain, ref.path = splitDomain(matches[1])
	ref.tag = matches[2]

	if matches[3] != "" {
		var err error
		ref.digest, err = digest.Parse(matches[3])
		if err != nil {
			return nil, err
		}
	}

	return &ref, nil
}
