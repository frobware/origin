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
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

type admissionTest struct {
	config     *testConfig
	attributes admission.Attributes
	handler    *AlwaysQualifyImages
	pod        *api.Pod
}

type testConfig struct {
	AdmissionDomain      string
	AdmissionObject      runtime.Object
	ContainersDomain     string
	Images               []string
	InitContainersDomain string
	Resource             string
	Subresource          string
}

func testImages() []string {
	return []string{
		"busybox",
		"busybox:latest",
		"foo/busybox",
		"foo/busybox:v1.2.3",
	}
}

func qualifyImage(domain, repo string) string {
	if domain == "" {
		return repo
	}
	return fmt.Sprintf("%s/%s", domain, repo)
}

func makeContainers(domain string, images []string) []api.Container {
	containers := make([]api.Container, len(images))

	for i := range images {
		containers[i] = api.Container{
			Name:  fmt.Sprintf("%v", i),
			Image: qualifyImage(domain, images[i]),
		}
	}

	return containers
}

func newTest(c *testConfig) admissionTest {
	pod := api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admissionTest",
			Namespace: "newAdmissionTest",
		},
		Spec: api.PodSpec{
			InitContainers: makeContainers(c.InitContainersDomain, c.Images),
			Containers:     makeContainers(c.ContainersDomain, c.Images),
		},
	}

	if c.AdmissionObject == nil {
		c.AdmissionObject = &pod
	}

	if c.Resource == "" {
		c.Resource = "pods"
	}

	if c.Images == nil {
		panic("bad test setup; no images specified")
	}

	attributes := admission.NewAttributesRecord(
		c.AdmissionObject,
		nil,
		api.Kind("Pod").WithVersion("version"),
		"Namespace",
		"Name",
		api.Resource(c.Resource).WithVersion("version"),
		c.Subresource,
		admission.Create, // XXX and update?
		nil)

	return admissionTest{
		attributes: attributes,
		config:     c,
		handler:    NewAlwaysQualifyImages(c.AdmissionDomain),
		pod:        &pod,
	}
}

func imageNames(containers []api.Container) []string {
	names := make([]string, len(containers))
	for i := range containers {
		names[i] = containers[i].Image
	}
	return names
}

func assertImageNamesEqual(t *testing.T, expected, actual []api.Container) {
	a, b := imageNames(expected), imageNames(actual)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
}

func TestAdmissionWhereInitContainersAreUnqualified(t *testing.T) {
	for i, domain := range []string{
		"test.io",
		"localhost",
		"localhost:5000",
		"a.b.c.d.e.f",
		"a.b.c.d.e.f:5000",
	} {
		for _, image := range testImages() {
			test := newTest(&testConfig{
				Images:           []string{image},
				AdmissionDomain:  domain,
				ContainersDomain: "someotherdomain.io",
			})

			if err := test.handler.Admit(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.AdmissionDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)

			if err := test.handler.Validate(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.AdmissionDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)
		}
	}
}

func TestAdmissionWhereContainersAreUnqualified(t *testing.T) {
	for i, domain := range []string{
		"test.io",
		"localhost",
		"localhost:5000",
		"a.b.c.d.e.f",
		"a.b.c.d.e.f:5000",
	} {
		for _, image := range testImages() {
			test := newTest(&testConfig{
				Images:               []string{image},
				AdmissionDomain:      domain,
				InitContainersDomain: "someotherdomain.io",
			})

			if err := test.handler.Admit(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.AdmissionDomain, test.config.Images), test.pod.Spec.Containers)

			if err := test.handler.Validate(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.AdmissionDomain, test.config.Images), test.pod.Spec.Containers)
		}
	}
}

func TestAdmissionWhereAllContainersAreAlreadyQualified(t *testing.T) {
	existingDomain := "someotherdomain.io"

	for i, domain := range []string{
		"test.io",
		"localhost",
		"localhost:5000",
		"a.b.c.d.e.f",
		"a.b.c.d.e.f:5000",
	} {
		for _, image := range testImages() {
			test := newTest(&testConfig{
				Images:               []string{image},
				AdmissionDomain:      domain,
				InitContainersDomain: existingDomain,
				ContainersDomain:     existingDomain,
			})

			if err := test.handler.Admit(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)

			if err := test.handler.Validate(test.attributes); err != nil {
				t.Fatalf("test %#v: unexpected error returned from admission handler: %s", i, err)
			}

			assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
			assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)
		}
	}
}

func TestAdmissionWhereExistingImageNameIsInvalid(t *testing.T) {
	for _, image := range testImages() {
		domain := "test.io"

		test := newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      domain,
			InitContainersDomain: "!bad.domain!",
			ContainersDomain:     domain,
		})

		if err := test.handler.Admit(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}

		test = newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      domain,
			InitContainersDomain: domain,
			ContainersDomain:     "!bad.domain!",
		})

		if err := test.handler.Admit(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}
	}
}

func TestAdmissionErrorsWithInvalidDomains(t *testing.T) {
	for _, image := range testImages() {
		domain := "test.io"

		// Test AdmissionDomain domain is invalid for InitContainers.

		test := newTest(&testConfig{
			Images:           []string{image},
			AdmissionDomain:  strings.Repeat("x", 255) + domain,
			ContainersDomain: domain,
		})

		if err := test.handler.Admit(test.attributes); err == nil {
			t.Errorf("expected error from admission handler")
		}

		// Test AdmissionDomain domain is invalid for Containers,

		test = newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      strings.Repeat("x", 255) + domain,
			InitContainersDomain: domain,
		})

		if err := test.handler.Admit(test.attributes); err == nil {
			t.Errorf("expected error from admission handler")
		}
	}
}

func TestAdmissionErrorsOnNonPodObject(t *testing.T) {
	test := newTest(&testConfig{
		Images:          testImages(),
		AdmissionDomain: "test.io",
		AdmissionObject: &api.ReplicationController{},
	})

	if err := test.handler.Admit(test.attributes); err == nil {
		t.Fatalf("expected an error from admission handler")
	}

	if err := test.handler.Validate(test.attributes); err == nil {
		t.Fatalf("expected an error from admission handler")
	}
}

func TestAdmissionIsIgnoredForSubresource(t *testing.T) {
	test := newTest(&testConfig{
		Images:          testImages(),
		AdmissionDomain: "test.io",
		Subresource:     "subresource",
	})

	// Not expecting an error for Admit() or Validate() because we
	// are operating on a subresource of pod. The handler will
	// ignore calls for these attributes and this means the
	// container names should remain unqualified.

	if err := test.handler.Admit(test.attributes); err != nil {
		t.Errorf("expected an error from admission handler")
	}

	assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
	assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)

	if err := test.handler.Validate(test.attributes); err != nil {
		t.Fatalf("expected an error from admission handler")
	}

	assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
	assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)
}

func TestAdmissionErrorsOnNonPodsResource(t *testing.T) {
	test := newTest(&testConfig{
		Images:          testImages(),
		AdmissionDomain: "test.io",
		Resource:        "nonpods",
	})

	if err := test.handler.Admit(test.attributes); err != nil {
		t.Fatalf("expected error from admission handler")
	}

	assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
	assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)

	if err := test.handler.Validate(test.attributes); err != nil {
		t.Fatalf("expected error from admission handler")
	}

	assertImageNamesEqual(t, makeContainers(test.config.InitContainersDomain, test.config.Images), test.pod.Spec.InitContainers)
	assertImageNamesEqual(t, makeContainers(test.config.ContainersDomain, test.config.Images), test.pod.Spec.Containers)
}

func TestValidateErrorsWhenImageNamesAreInvalid(t *testing.T) {
	for _, image := range testImages() {
		test := newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      "test.io",
			InitContainersDomain: "!bad.domain!",
		})

		if err := test.handler.Validate(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}

		test = newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      "test.io",
			InitContainersDomain: "", // unqualified
		})

		if err := test.handler.Validate(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}

		// Same tests, but for Containers.

		test = newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      "test.io",
			InitContainersDomain: "test.io",
			ContainersDomain:     "!bad.domain!",
		})

		if err := test.handler.Validate(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}

		test = newTest(&testConfig{
			Images:               []string{image},
			AdmissionDomain:      "test.io",
			InitContainersDomain: "test.io", // unqualified
			ContainersDomain:     "",        // unqualified
		})

		if err := test.handler.Validate(test.attributes); err == nil {
			t.Fatalf("expected error from admission handler")
		}
	}
}
