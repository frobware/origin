package imagequalify

import (
	"fmt"
	"io"
	"sort"

	"github.com/golang/glog"
	configlatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/image/admission/imagequalify/api"
	"github.com/openshift/origin/pkg/image/admission/imagequalify/api/validation"
)

func readConfig(rdr io.Reader) (*api.ImageQualifyConfig, error) {
	obj, err := configlatest.ReadYAML(rdr)
	if err != nil {
		glog.V(5).Infof("%s error reading config: %v", api.PluginName, err)
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	config, ok := obj.(*api.ImageQualifyConfig)
	if !ok {
		return nil, fmt.Errorf("unexpected config object: %#v", obj)
	}
	glog.V(5).Infof("%s config is: %#v", api.PluginName, config)
	if errs := validation.Validate(config); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}
	sort.Stable(ByPatternPriority(config.Rules))
	return config, nil
}
