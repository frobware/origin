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
		// This case is special. Ordinarily things must look
		// like a domain name, or localhost, and/or have a
		// port number. For patterns we look to see how many
		// other delimiters there are and, if there's more
		// than one, the first must represent the desired
		// domain.
		n := strings.Count(image, "/")
		if n > 1 {
			return image[:i], image[i+1:]
		}
		return "", image
	}
	return image[:i], image[i+1:]
}

func parsePattern(s string) (*pattern, error) {
	matches := ReferenceRegexp.FindStringSubmatch(s)

	if matches == nil {
		if s == "" {
			return nil, reference.ErrNameEmpty
		}
		if ReferenceRegexp.FindStringSubmatch(strings.ToLower(s)) != nil {
			return nil, reference.ErrNameContainsUppercase
		}
		return nil, reference.ErrReferenceInvalidFormat
	}

	var p pattern

	p.domain, p.path = splitDomain(matches[1])
	p.tag = matches[2]

	if matches[3] != "" {
		var err error
		p.digest, err = digest.Parse(matches[3])
		if err != nil {
			return nil, err
		}
	}

	return &p, nil
}
