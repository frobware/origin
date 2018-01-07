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

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/openshift/origin/pkg/image/qualify"
)

func ruleError(filename string, e *qualify.RuleError) string {
	if e.InvalidDomain != "" {
		return fmt.Sprintf("%q:%v: invalid domain %q: %s", filename, e.LineNumber, e.InvalidDomain, e.Message)
	} else if e.InvalidPattern != "" {
		return fmt.Sprintf("%q:%v: invalid pattern %q: %s", filename, e.LineNumber, e.InvalidPattern, e.Message)
	} else {
		return fmt.Sprintf("%q:%v: %q: %s", filename, e.LineNumber, e.Definition, e.Message)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) != 3 {
		log.Fatalf("usage: <filename> <image>")
	}

	filename, imageref := os.Args[1], os.Args[2]
	domain, _, err := qualify.SplitImageName(imageref)

	if err != nil {
		log.Fatalf("%q is an invalid image reference: %q", imageref, err.Error())
	}

	if domain != "" {
		log.Fatalf("%q already has a domain component", imageref)
	}

	rules, err := qualify.LoadRules(filename)

	if err != nil {
		if v, ok := err.(*qualify.RuleError); ok {
			log.Fatalf(ruleError(filename, v))
		}
		log.Fatalf("error loading rules %q: %s", filename, err)
	}

	_, qualifiedImage := qualify.Qualify(imageref, rules)

	if qualifiedImage == "" {
		qualifiedImage = imageref
	}

	fmt.Println(qualifiedImage)
}
