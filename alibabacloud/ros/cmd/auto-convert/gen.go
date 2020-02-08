package main

import (
	"encoding/json"
	"errors"
	"fmt"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	app := &cli.App{
		Name:  "gen",
		Usage: "generate components from ROS resource types",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "access_key_id",
				Aliases:  []string{"i"},
				Usage:    "Specify access key id",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "access_key_secret",
				Aliases:  []string{"s"},
				Usage:    "Specify access key secret",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "resource_type",
				Aliases:  []string{"r"},
				Usage:    "generate component from specific ROS resource type",
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			gen(c.String("access_key_id"), c.String("access_key_secret"), c.String("resource_type"))
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func gen(accessKeyId, accessKeySecret, resourceType string) {

	rosClient, err := rosapi.NewClientWithAccessKey(
		"cn-hangzhou",
		accessKeyId,
		accessKeySecret)
	if err != nil {
		panic(err)
	}
	request := rosapi.CreateListResourceTypesRequest()
	response, err := rosClient.ListResourceTypes(request)
	if err != nil {
		panic(err)
	}

	var resourceTypes []string
	if resourceType != "" {
		exits := true
		for _, resType := range response.ResourceTypes {
			if resType == resourceType {
				exits = true
			}
		}
		if exits == false {
			panic(errors.New(resourceType + " not exists"))
		}
		resourceTypes = []string{resourceType}
	} else {
		resourceTypes = response.ResourceTypes
	}

	dir := "workloads"
	err = os.Mkdir(dir, os.ModePerm)
	if err == nil {
		fmt.Printf("Make dir %s \n", dir)
	}

	for _, resType := range resourceTypes {
		fmt.Println("Generating " + resType)
		// Get workloadType
		workloadType := getWorkloadType(resType, rosClient)
		workloadTypeYaml, err := yaml.Marshal(workloadType)
		if err != nil {
			panic(err)
		}
		// Write to file
		filePath := filepath.Join(dir, workloadType.Spec.Names.Kind+".yml")
		if ioutil.WriteFile(filePath, workloadTypeYaml, 0644) == nil {
			fmt.Println("Write to " + filePath)
		}

		time.Sleep(time.Millisecond * 200)
	}
}

func getWorkloadType(resourceType string, rosClient *rosapi.Client) WorkloadType {
	request := rosapi.CreateGetResourceTypeRequest()
	request.ResourceType = resourceType
	response, err := rosClient.GetResourceType(request)
	if err != nil {
		serverErr, ok := err.(*sdkerrors.ServerError)
		if ok && serverErr.ErrorCode() == "Throttling.User" {
			time.Sleep(time.Second)
			return getWorkloadType(resourceType, rosClient)
		}
		panic(err)
	}

	schema := getObjectJsonSchema(response.Properties, true)
	workloadSettings, _ := json.MarshalIndent(schema, "", "  ")

	nameKind := strings.ReplaceAll(response.ResourceType, "ALIYUN::", "")
	nameKind = strings.ReplaceAll(nameKind, "::", "_")
	workloadType := WorkloadType{
		ApiVersion: "core.oam.dev/v1alpha1",
		Kind:       "WorkloadType",
		Metadata: Metadata{
			Name: strings.Replace(strings.ToLower(nameKind), "_", "-", 1),
		},
		Spec: Spec{
			Group:            "ros.aliyun.com",
			Version:          "v1alpha1",
			Names:            Names{Kind: nameKind},
			WorkloadSettings: string(workloadSettings),
		},
	}

	return workloadType
}

func getObjectJsonSchema(rosProperties map[string]interface{}, withSchemaUrl bool) JsonSchema {
	var requiredParameters []string
	properties := make(map[string]interface{})

	for name, schema := range rosProperties {
		schema, _ := schema.(map[string]interface{})

		required, ok := schema["Required"].(bool)
		if !ok {
			required = false
		}
		if required {
			requiredParameters = append(requiredParameters, name)
		}

		properties[name] = getJsonSchema(schema)
	}

	objectProperty := JsonSchema{
		Type:       "object",
		Required:   requiredParameters,
		Properties: properties,
	}

	if withSchemaUrl {
		objectProperty.Schema = "http://json-schema.org/draft-07/schema#"
	}

	return objectProperty
}

func getJsonSchema(schema map[string]interface{}) JsonSchema {
	description, ok := schema["Description"].(string)
	if !ok {
		description = ""
	}
	fieldType, ok := schema["Type"].(string)
	if !ok {
		return JsonSchema{
			Description: description,
		}
	}
	fieldType = transType(fieldType)
	valueDefault, _ := schema["Default"]
	constraintsJsonStr, _ := json.Marshal(schema["Constraints"])
	constraints := make([]map[string]interface{}, 0)
	_ = json.Unmarshal(constraintsJsonStr, &constraints)
	enum, numRange, length, pattern := getConstraints(constraints)

	switch fieldType {
	case "number":
		numberProperty := JsonSchema{
			Type:        "number",
			Description: description,
		}
		if numRange != nil {
			numberProperty.Minimum = numRange["Min"]
			numberProperty.Maximum = numRange["Max"]
		}
		numberProperty.Default = valueDefault
		numberProperty.Description = description
		numberProperty.Enum = enum
		return numberProperty
	case "integer":
		integerProperty := JsonSchema{
			Type:        "integer",
			Description: description,
		}
		if numRange != nil {
			integerProperty.Minimum = numRange["Min"]
			integerProperty.Maximum = numRange["Max"]
		}
		integerProperty.Default = valueDefault
		integerProperty.Description = description
		integerProperty.Enum = enum
		return integerProperty
	case "string":
		stringProperty := JsonSchema{
			Type:        "string",
			Description: description,
		}
		if length != nil {
			stringProperty.MinLength = length["Min"]
			stringProperty.MaxLength = length["Max"]
		}
		stringProperty.Default = valueDefault
		stringProperty.Description = description
		stringProperty.Enum = enum
		stringProperty.Pattern = pattern
		return stringProperty
	case "array":
		var items JsonSchema
		schema_, ok := schema["Schema"].(map[string]interface{})
		if ok {
			subSchema := schema_["*"].(map[string]interface{})
			items = getJsonSchema(subSchema)
		}

		arrayProperty := JsonSchema{
			Type:        "array",
			Description: description,
			Items:       items,
		}
		if length != nil {
			arrayProperty.MinItems = length["Min"]
			arrayProperty.MaxItems = length["Max"]
		}
		arrayProperty.Default = valueDefault
		arrayProperty.Description = description
		return arrayProperty
	case "object":
		var objectProperty JsonSchema
		rawProperties, ok := schema["Schema"].(map[string]interface{})
		if ok {
			objectProperty = getObjectJsonSchema(rawProperties, false)
		}

		if length != nil {
			objectProperty.MinProperties = length["Min"]
			objectProperty.MaxProperties = length["Max"]
		}
		return objectProperty
	case "boolean":
		booleanProperty := JsonSchema{
			Type:        "boolean",
			Description: description,
		}
		booleanProperty.Default = valueDefault
		booleanProperty.Description = description
		booleanProperty.Enum = enum
		return booleanProperty
	default:
		panic(errors.New("Unsupported type: " + fieldType))
	}
}

func transType(fieldType string) string {
	switch fieldType {
	case "list":
		return "array"
	case "map":
		return "object"
	default:
		return fieldType
	}
}

func getConstraints(constraints []map[string]interface{}) ([]interface{}, map[string]float64, map[string]int, string) {
	var enum []interface{}
	numRange := make(map[string]float64)
	length := make(map[string]int)
	var pattern string

	if constraints != nil {
		for _, constraint := range constraints {
			if enum_, ok := constraint["AllowedValues"]; ok {
				enum = enum_.([]interface{})
				continue
			}
			if numRange_, ok := constraint["Range"]; ok {
				for k, v := range numRange_.(map[string]interface{}) {
					numRange[k] = v.(float64)
				}
				continue
			}
			if length_, ok := constraint["Length"]; ok {
				for k, v := range length_.(map[string]interface{}) {
					length[k] = int(v.(float64))
				}
				continue
			}
			if pattern_, ok := constraint["AllowedPattern"]; ok {
				pattern = pattern_.(string)
				continue
			}
		}
	}

	return enum, numRange, length, pattern
}

type WorkloadType struct {
	ApiVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

type Metadata struct {
	Name string `json:"name"`
}

type Spec struct {
	Group            string `json:"group"`
	Version          string `json:"version"`
	Names            Names  `json:"names"`
	WorkloadSettings string `json:"workloadSettings" yaml:"workloadSettings"`
}

type Names struct {
	Kind string `json:"kind"`
}

type JsonSchema struct {
	Schema        string                 `json:"$schema,omitempty"`
	Type          string                 `json:"type,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Required      []string               `json:"required,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	Default       interface{}            `json:"default,omitempty"`
	Enum          []interface{}          `json:"Enum,omitempty"`
	Items         interface{}            `json:"items,omitempty"`
	MinItems      int                    `json:"minItems,omitempty"`
	MaxItems      int                    `json:"maxItems,omitempty"`
	MinLength     int                    `json:"minLength,omitempty"`
	MaxLength     int                    `json:"maxLength,omitempty"`
	MinProperties int                    `json:"minProperties,omitempty"`
	MaxProperties int                    `json:"maxProperties,omitempty"`
	Pattern       string                 `json:"pattern,omitempty"`
	Minimum       float64                `json:"minimum,omitempty"`
	Maximum       float64                `json:"maximum,omitempty"`
}
