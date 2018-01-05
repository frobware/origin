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
)

func ruleError(e *qualify.RuleError) string {
	return fmt.Sprintf("%q:%v: %q: %s", e.Filename, e.LineNum, e.Line, e.Message)
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

	rules, err := qualify.ParseRules(filename)

	if err != nil {
		if v, ok := err.(*qualify.RuleError); ok {
			log.Fatalf("error loading rules: %s", ruleError(v))
		}
		log.Fatalf("error loading rules %q: %s", filename, err)
	}

	_, qualifiedImage := qualify.Qualify(imageref, rules)

	if qualifiedImage == "" {
		fmt.Printf("No match for %q\n", imageref)
		os.Exit(1)
	}

	fmt.Println(qualifiedImage)
}
