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
	"fmt"
	"io"
	"io/ioutil"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	alwaysqualifyimagesapi "github.com/openshift/origin/pkg/image/admission/alwaysqualifyimages/apis/alwaysqualifyimages"
)

var (
	codecs = serializer.NewCodecFactory(runtime.NewScheme())
)

func loadConfiguration(config io.Reader) (*alwaysqualifyimagesapi.Configuration, error) {
	// if no config is provided, return a default configuration
	if config == nil {
		return &alwaysqualifyimagesapi.Configuration{}, nil
	}
	// we have a config so parse it.
	data, err := ioutil.ReadAll(config)
	if err != nil {
		return nil, err
	}
	decoder := codecs.UniversalDecoder()
	decodedObj, err := runtime.Decode(decoder, data)
	if err != nil {
		return nil, err
	}
	alwaysqualifyimagesConfiguration, ok := decodedObj.(*alwaysqualifyimagesapi.Configuration)
	if !ok {
		return nil, fmt.Errorf("unexpected type: %T", decodedObj)
	}
	return alwaysqualifyimagesConfiguration, nil
}
