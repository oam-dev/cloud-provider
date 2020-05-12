package ros

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

var template = &Template{
	ROSTemplateFormatVersion: "2015-09-01",
	Parameters:               map[string]Parameter{},
	Resources: map[string]Resource{
		"Vpc": {
			Type: "ALIYUN::ECS::VPC",
			Properties: map[string]interface{}{
				"VpcName":     "MyVpc",
				"CidrBlock":   "192.168.0.0/16",
				"Description": "My VPC",
			},
			DependsOn:      []string{},
			DeletionPolicy: "Retain",
		}},
	Outputs: map[string]Output{},
}

var templateBodyBytes, _ = json.Marshal(template)
var templateBody = string(templateBodyBytes)

func TestNewStack(t *testing.T) {
	type args struct {
		appContext *appconf.Context
		stackName  string
	}
	tests := []struct {
		name          string
		args          args
		dryRunHandler func(stack *Stack, request requests.AcsRequest) error
		wantStackId   string
	}{
		{
			name: "TestNormal",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			dryRunHandler: func(stack *Stack, request requests.AcsRequest) error {
				println("Dry run")
				switch req := request.(type) {
				case *rosapi.CreateStackRequest:
					assert.Equal(t, config.RosCtrlConf.UserAgent, req.GetUserAgent()["Service"])
					assert.Equal(t, requests.Integer("60"), req.TimeoutInMinutes)
					assert.Equal(t, "MyStack", req.StackName)
					assert.Equal(t, requests.Boolean("false"), req.DisableRollback)
					assert.Equal(t, []rosapi.CreateStackParameters{}, *req.Parameters)
					assert.Equal(t, templateBody, req.TemplateBody)

				default:
					assert.Fail(t, "request type error")
				}
				return nil
			},
			wantStackId: DryRunFakeStack,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, _ := NewStack(
				tt.args.appContext,
				tt.args.stackName,
				template,
				WithDryRunHandler(tt.dryRunHandler),
			)
			assert.Equal(t, tt.wantStackId, stack.Id)
		})
	}
}

func TestStack_Update(t *testing.T) {
	type args struct {
		appContext *appconf.Context
		stackName  string
	}
	tests := []struct {
		name          string
		args          args
		dryRunHandler func(stack *Stack, request requests.AcsRequest) error
	}{
		{
			name: "TestNormal",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			dryRunHandler: func(stack *Stack, request requests.AcsRequest) error {
				println("Dry run")
				switch req := request.(type) {
				case *rosapi.CreateStackRequest:
					return nil
				case *rosapi.UpdateStackRequest:
					assert.Equal(t, config.RosCtrlConf.UserAgent, req.GetUserAgent()["Service"])
					assert.Equal(t, []rosapi.UpdateStackParameters{}, *req.Parameters)
					assert.Equal(t, templateBody, req.TemplateBody)
				default:
					assert.Fail(t, "request type error")
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, _ := NewStack(
				tt.args.appContext,
				tt.args.stackName,
				template,
				WithDryRunHandler(tt.dryRunHandler),
			)
			err := stack.Update(template)
			assert.Nil(t, err)
		})
	}
}

func TestStack_Delete(t *testing.T) {
	type args struct {
		appContext *appconf.Context
		stackName  string
	}
	tests := []struct {
		name          string
		args          args
		dryRunHandler func(stack *Stack, request requests.AcsRequest) error
	}{
		{
			name: "TestNormal",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			dryRunHandler: func(stack *Stack, request requests.AcsRequest) error {
				println("Dry run")
				switch req := request.(type) {
				case *rosapi.CreateStackRequest:
					return nil
				case *rosapi.DeleteStackRequest:
					assert.Equal(t, config.RosCtrlConf.UserAgent, req.GetUserAgent()["Service"])
				default:
					assert.Fail(t, "request type error")
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, _ := NewStack(
				tt.args.appContext,
				tt.args.stackName,
				template,
				WithDryRunHandler(tt.dryRunHandler),
			)
			err := stack.Delete()
			assert.Nil(t, err)
		})
	}
}

func TestStack_Refresh(t *testing.T) {
	type args struct {
		appContext *appconf.Context
		stackName  string
	}
	tests := []struct {
		name          string
		args          args
		dryRunHandler func(stack *Stack, request requests.AcsRequest) error
		wantStatus    string
	}{
		{
			name: "TestNormal",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			dryRunHandler: func(stack *Stack, request requests.AcsRequest) error {
				println("Dry run")
				switch req := request.(type) {
				case *rosapi.CreateStackRequest:
					return nil
				case *rosapi.GetStackRequest:
					stack.Status = string(CreateComplete)
					assert.Equal(t, config.RosCtrlConf.UserAgent, req.GetUserAgent()["Service"])
				default:
					assert.Fail(t, "request type error")
				}
				return nil
			},
			wantStatus: string(CreateComplete),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, _ := NewStack(
				tt.args.appContext,
				tt.args.stackName,
				template,
				WithDryRunHandler(tt.dryRunHandler),
			)
			assert.Equal(t, stack.Status, "")
			err := stack.Refresh()
			assert.Nil(t, err)
			assert.Equal(t, stack.Status, tt.wantStatus)
		})
	}
}

func TestStack_WaitUntilDone(t *testing.T) {
	type args struct {
		appContext *appconf.Context
		stackName  string
	}
	tests := []struct {
		name              string
		args              args
		changeStackStatus func(stack *Stack)
		isSuccess         bool
		wantStatusReason  string
	}{
		{
			name: "TestSuccess",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			changeStackStatus: func(stack *Stack) {
				stack.Status = string(CheckComplete)
			},
			isSuccess:        true,
			wantStatusReason: "",
		},
		{
			name: "TestFailed",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				stackName:  "MyStack",
			},
			changeStackStatus: func(stack *Stack) {
				stack.Status = string(RollbackComplete)
				stack.StatusReason = "Rollback"
			},
			isSuccess:        false,
			wantStatusReason: "Rollback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.RosCtrlConf.StackCheckInterval = 1
			stack, _ := NewStack(
				tt.args.appContext,
				tt.args.stackName,
				template,
			)
			go tt.changeStackStatus(stack)
			isSuccess, statusReason := stack.WaitUntilDone()
			assert.Equal(t, tt.isSuccess, isSuccess)
			assert.Equal(t, tt.wantStatusReason, statusReason)
		})
	}
}
