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
	"fmt"

	"github.com/golang/glog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/util/parsers"
	"github.com/openshift/origin/pkg/image/qualify"
)

// AlwaysQualifyImages is an implementation of admission.Interface. It
// looks at all new pods and overrides any container's image name that
// is unqualified and adds domain.
type AlwaysQualifyImages struct {
	*admission.Handler
	domain string
}

var _ admission.MutationInterface = &AlwaysQualifyImages{}
var _ admission.ValidationInterface = &AlwaysQualifyImages{}

// qualifyContainerImages modifies containers to include domain iff
// the image name is unqualified (i.e., has no domain). It fails fast
// if adding domain results in an invalid image.
func qualifyContainerImages(domain string, containers []api.Container) (string, error) {
	for i := range containers {
		d, _, err := qualify.SplitImageName(containers[i].Image)
		if err != nil {
			return containers[i].Image, err
		}
		if d != "" {
			glog.V(2).Infof("not qualifying image %q as it has a domain", containers[i].Image)
			continue
		}
		newName := domain + "/" + containers[i].Image
		if _, _, _, err := parsers.ParseImageName(newName); err != nil {
			return newName, err
		}
		glog.V(2).Infof("qualifying image %q as %q", containers[i].Image, newName)
		containers[i].Image = newName
	}
	return "", nil
}

// Admit makes an admission decision based on the request attributes.
// If the attributes are valid then any container image names that are
// unqualified (i.e., have no domain component) will be qualified with
// domain.
func (a *AlwaysQualifyImages) Admit(attributes admission.Attributes) error {
	// Ignore all calls to subresources or resources other than pods.
	if shouldIgnore(attributes) {
		return nil
	}

	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	if image, err := qualifyContainerImages(a.domain, pod.Spec.InitContainers); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid image name %q: %s", image, err.Error()))
	}

	if image, err := qualifyContainerImages(a.domain, pod.Spec.Containers); err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid image name %q: %s", image, err.Error()))
	}

	return nil
}

// Validate makes sure that all images names in a POD spec have a
// domain component as part of their image name.
func (a *AlwaysQualifyImages) Validate(attributes admission.Attributes) error {
	if shouldIgnore(attributes) {
		return nil
	}

	pod, ok := attributes.GetObject().(*api.Pod)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	for _, container := range pod.Spec.InitContainers {
		domain, _, err := qualify.SplitImageName(container.Image)
		if err != nil {
			return err
		}
		if domain == "" {
			return errors.New("image has no domain")
		}
	}

	for _, container := range pod.Spec.Containers {
		domain, _, err := qualify.SplitImageName(container.Image)
		if err != nil {
			return err
		}
		if domain == "" {
			return errors.New("image has no domain")
		}
	}

	return nil
}

func isSubresourceRequest(attributes admission.Attributes) bool {
	return len(attributes.GetSubresource()) > 0
}

func isPodsRequest(attributes admission.Attributes) bool {
	return attributes.GetResource().GroupResource() == api.Resource("pods")
}

func shouldIgnore(attributes admission.Attributes) bool {
	switch {
	case isSubresourceRequest(attributes):
		return true
	case !isPodsRequest(attributes):
		return true
	default:
		return false
	}
}

// NewAlwaysQualifyImages creates a new admission control handler that
// handles Create and Update operations and will add domain to
// unqualified Pod container image names.
func NewAlwaysQualifyImages(domain string) *AlwaysQualifyImages {
	return &AlwaysQualifyImages{
		Handler: admission.NewHandler(admission.Create, admission.Update),
		domain:  domain,
	}
}
