package ros

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/component"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
)

type Template struct {
	ROSTemplateFormatVersion string               `json:"ROSTemplateFormatVersion"`
	Description              string               `json:"Description,omitempty"`
	Parameters               map[string]Parameter `json:"Parameters,omitempty"`
	Resources                map[string]Resource  `json:"Resources,omitempty"`
	Outputs                  map[string]Output    `json:"Outputs,omitempty"`
}

type Parameter struct {
	Name        string      `json:"-"`
	Type        string      `json:"Type"`
	Value       string      `json:"-"`
	Default     interface{} `json:"Default,omitempty"`
	Description string      `json:"Description,omitempty"`
}

type Resource struct {
	Type           string                 `json:"Type"`
	Properties     map[string]interface{} `json:"Properties,omitempty"`
	DependsOn      []string               `json:"DependsOn,omitempty"`
	DeletionPolicy string                 `json:"DeletionPolicy,omitempty"`
}

type Output struct {
	Description string      `json:"Description,omitempty"`
	Value       interface{} `json:"Value,omitempty"`
}

// NewTemplate parses application configuration and returns ROS template
func NewTemplate(rosContext *Context, appConf *rosv1alpha1.ApplicationConfiguration) (*Template, error) {
	template := Template{
		ROSTemplateFormatVersion: "2015-09-01",
		Parameters:               make(map[string]Parameter),
		Resources:                make(map[string]Resource),
		Outputs:                  make(map[string]Output),
	}
	for _, compConf := range appConf.Spec.Components {
		// get component
		instanceName := compConf.InstanceName
		compSchematic, err := component.Get(appConf.Namespace, compConf.ComponentName)
		if err != nil {
			return nil, err
		}

		workloadType := compSchematic.Spec.WorkloadType
		logging.Default.Info(
			"Convert component to ROS template",
			"ComponentName", compConf.ComponentName,
			"WorkloadType", workloadType,
		)

		// get resource type
		resourceType, err := getResourceType(workloadType)
		if err != nil {
			logging.Default.Error(
				err, "Ignore component due to invalid workload type",
				"ComponentName", compConf.ComponentName,
				"WorkloadType", workloadType,
			)
			continue
		}

		// generate template parts
		err = template.genParametersAndResource(resourceType, compConf, compSchematic.Spec, appConf.Spec.Components)
		if err != nil {
			return nil, err
		}
		// TODO(Prodesire): dry run mode also need to add outputs
		if !rosContext.DryRun {
			// get resource type detail
			request := rosapi.CreateGetResourceTypeRequest()
			request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
			request.ResourceType = resourceType
			response, err := rosContext.RosClient.GetResourceType(request)
			if err != nil {
				return nil, err
			}
			template.genOutputs(instanceName, response.Attributes)
		}
	}

	return &template, nil
}

// Marshal generates JSON string
func (t *Template) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

// genParametersAndResources generates Parameters and Resource in template
func (t *Template) genParametersAndResource(
	resourceType string,
	compConf v1alpha1.ComponentConfiguration,
	compSpec v1alpha1.ComponentSpec,
	compConfs []v1alpha1.ComponentConfiguration,
) error {
	resource := Resource{
		Type:       resourceType,
		Properties: make(map[string]interface{}),
		DependsOn:  make([]string, 0),
	}

	// compSpec workload settings to ROS Properties
	properties := make(map[string]interface{})
	err := json.Unmarshal(compSpec.WorkloadSettings.Raw, &properties)
	if err != nil {
		return err
	}

	for name, value := range properties {
		resource.Properties[name] = value
	}

	// compConf parameterValues to ROS Properties
	for _, ParameterValue := range compConf.ParameterValues {
		name := ParameterValue.Name
		value := ParameterValue.Value
		valueFrom := ParameterValue.From

		// if supply from, use it for its high priority
		if valueFrom != nil {
			refCompInstanceName := valueFrom.Component
			refField := valueFrom.FieldPath
			if strings.HasPrefix(refField, ".status.") {
				refField = strings.Replace(refField, ".status.", "", 1)
				if strings.Contains(refField, ".") {
					return errors.New(fmt.Sprintf("Invalid fieldPath '%s' does not meet format (.status.{FieldName})", valueFrom.FieldPath))
				}
			}

			// check whether ref comp exists
			found := false
			for _, conf := range compConfs {
				if conf.InstanceName == refCompInstanceName {
					found = true
				}
			}
			if !found {
				return errors.New(fmt.Sprintf("Invalid reference '%s' which refers a no exist component instance", refCompInstanceName))
			}

			// check ref self
			if refCompInstanceName == compConf.InstanceName {
				return errors.New(fmt.Sprintf("Invalid reference '%s' which refers to component instance itself", refCompInstanceName))
			}

			// set property
			resource.Properties[name] = map[string][]string{"Fn::GetAtt": {refCompInstanceName, refField}}

			// set ROS DependsOn
			found = false
			for _, dependOn := range resource.DependsOn {
				if dependOn == refCompInstanceName {
					found = true
				}
			}
			if !found {
				resource.DependsOn = append(resource.DependsOn, refCompInstanceName)
			}
		} else if value != "" {
			resource.Properties[name] = value
		} else {
			return errors.New(fmt.Sprintf("Either value or from should be supplied from parameterValues"))
		}
	}

	// Resource DeletionPolicy
	for _, trait := range compConf.Traits {
		if trait.Name != "DeletionPolicy" || trait.Properties.Raw == nil {
			continue
		}

		props := make(map[string]string)
		err := json.Unmarshal(trait.Properties.Raw, &props)
		if err != nil {
			return err
		}

		policy, _ := props["policy"]
		if policy == "Retain" || policy == "Delete" {
			resource.DeletionPolicy = policy
		}
	}

	// set resource
	logicalId := compConf.InstanceName
	t.Resources[logicalId] = resource

	return nil
}

// genOutputs generates Outputs in template
func (t *Template) genOutputs(instanceName string, resourceAttributes map[string]interface{}) {
	logicalId := instanceName
	for name, attribute := range resourceAttributes {
		attribute := attribute.(map[string]interface{})
		description := attribute["Description"].(string)
		outputName := logicalId + "." + name
		output := Output{
			Description: description,
			Value:       map[string][2]string{"Fn::GetAtt": {logicalId, name}},
		}
		t.Outputs[outputName] = output
	}
}

// getResourceType gets ROS resource type from workloadType
func getResourceType(workloadType string) (string, error) {
	fmtmsg := "workloadType must be format of {group}/{version}.{type}"
	split := strings.Split(workloadType, "/")
	if len(split) != 2 {
		return "", errors.New(fmtmsg)
	}

	group := split[0]
	if group != config.ROS_GROUP {
		return "", errors.New(fmt.Sprintf("Group %s in workloadType is not supported", group))
	}

	split = strings.Split(split[1], ".")
	if len(split) != 2 {
		return "", errors.New(fmtmsg)
	}

	version := split[0]
	if version != "v1alpha1" {
		return "", errors.New(fmt.Sprintf("Version %s in workloadType is not supported", version))
	}

	split = strings.Split(split[1], "_")
	if len(split) != 2 {
		return "", errors.New("{type} in workloadType must be format of {product}_{restype}")
	}

	resourceType := fmt.Sprintf("ALIYUN::%s::%s", split[0], split[1])
	return resourceType, nil
}
