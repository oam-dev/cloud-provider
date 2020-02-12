package ros

import (
	"encoding/json"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"time"
)

const (
	StackId     = "StackId"
	StackName   = "StackName"
	StackStatus = "StackStatus"
	OutputKey   = "OutputKey"
	OutputValue = "OutputValue"
)

type Stack struct {
	Id           string                   `json:"Id"`
	Name         string                   `json:"Name"`
	Status       string                   `json:"Status"`
	StatusReason string                   `json:"StatusReason"`
	Outputs      []map[string]interface{} `json:"Outputs"`
}

type StackStatusType string

const (
	CreateInProgress         StackStatusType = "CREATE_IN_PROGRESS"
	CreateFailed             StackStatusType = "CREATE_FAILED"
	CreateComplete           StackStatusType = "CREATE_COMPLETE"
	UpdateInProgress         StackStatusType = "UPDATE_IN_PROGRESS"
	UpdateFailed             StackStatusType = "UPDATE_FAILED"
	UpdateComplete           StackStatusType = "UPDATE_COMPLETE"
	DeleteInProgress         StackStatusType = "DELETE_IN_PROGRESS"
	DeleteFailed             StackStatusType = "DELETE_FAILED"
	DeleteComplete           StackStatusType = "DELETE_COMPLETE"
	CreateRollbackInProgress StackStatusType = "CREATE_ROLLBACK_IN_PROGRESS"
	CreateRollbackFailed     StackStatusType = "CREATE_ROLLBACK_FAILED"
	CreateRollbackComplete   StackStatusType = "CREATE_ROLLBACK_COMPLETE"
	RollbackInProgress       StackStatusType = "ROLLBACK_IN_PROGRESS"
	RollbackFailed           StackStatusType = "ROLLBACK_FAILED"
	RollbackComplete         StackStatusType = "ROLLBACK_COMPLETE"
	CheckInProgress          StackStatusType = "CHECK_IN_PROGRESS"
	CheckFailed              StackStatusType = "CHECK_FAILED"
	CheckComplete            StackStatusType = "CHECK_COMPLETE"
	ReviewInProgress         StackStatusType = "REVIEW_IN_PROGRESS"
)

// NewStack parses application configuration and returns ROS template
func NewStack(rosContext *Context, stackName string, template *Template) (stack *Stack, err error) {
	// template body
	templateBody, err := json.Marshal(template)
	if err != nil {
		return
	}

	// parameters
	parameters := make([]rosapi.CreateStackParameters, 0)
	for _, param := range template.Parameters {
		stackParameter := rosapi.CreateStackParameters{
			ParameterKey:   param.Name,
			ParameterValue: param.Value,
		}
		parameters = append(parameters, stackParameter)
	}

	// create stack
	request := rosapi.CreateCreateStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.TimeoutInMinutes = "60"
	request.StackName = stackName
	request.DisableRollback = "false"
	request.Parameters = &parameters
	request.TemplateBody = string(templateBody)

	response, err := rosContext.RosClient.CreateStack(request)
	if err != nil {
		return
	}

	stack = &Stack{
		Id:   response.StackId,
		Name: stackName,
	}
	return
}

func (stack *Stack) Update(rosContext *Context, template *Template) error {
	// template body
	templateBody, err := json.Marshal(template)
	if err != nil {
		return err
	}

	// parameters
	parameters := make([]rosapi.UpdateStackParameters, 0)
	for _, param := range template.Parameters {
		stackParameter := rosapi.UpdateStackParameters{
			ParameterKey:   param.Name,
			ParameterValue: param.Value,
		}
		parameters = append(parameters, stackParameter)
	}

	// update stack
	request := rosapi.CreateUpdateStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.StackId = stack.Id
	request.Parameters = &parameters
	request.TemplateBody = string(templateBody)

	_, err = rosContext.RosClient.UpdateStack(request)
	if err != nil {
		return err
	}

	return nil
}

func (stack *Stack) Delete(rosContext *Context) error {
	request := rosapi.CreateDeleteStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.StackId = stack.Id

	_, err := rosContext.RosClient.DeleteStack(request)
	if err != nil {
		return err
	}

	return nil
}

func (stack *Stack) Refresh(rosContext *Context) error {
	request := rosapi.CreateGetStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.StackId = stack.Id

	resp, err := rosContext.RosClient.GetStack(request)
	if err != nil {
		return err
	}

	stack.Name = resp.StackName
	stack.Status = resp.Status
	stack.StatusReason = resp.StatusReason
	stack.Outputs = resp.Outputs

	return nil
}

func (stack *Stack) WaitUntilDone(rosContext *Context) (success bool, statusReason string) {
	for {
		time.Sleep(5 * time.Second)
		err := stack.Refresh(rosContext)
		if err != nil {
			logging.Default.Error(err, "Refresh stack error", StackId, stack.Id, StackName, stack.Name)
			continue
		}

		logging.Default.Info("Stack info", StackId, stack.Id, StackName, stack.Name, StackStatus, stack.Status)

		switch StackStatusType(stack.Status) {
		// success
		case CreateComplete:
			fallthrough
		case UpdateComplete:
			fallthrough
		case DeleteComplete:
			fallthrough
		case CheckComplete:
			logging.Default.Info("Stack check done", StackId, stack.Id, StackName, stack.Name, StackStatus, stack.Status)
			return true, ""

		// fail
		case CreateFailed:
			fallthrough
		case UpdateFailed:
			fallthrough
		case DeleteFailed:
			fallthrough
		case CheckFailed:
			fallthrough
		case CreateRollbackFailed:
			fallthrough
		case CreateRollbackComplete:
			fallthrough
		case RollbackFailed:
			fallthrough
		case RollbackComplete:
			logging.Default.Info("Stack check failed", StackId, stack.Id, StackName, stack.Name, StackStatus, stack.Status)
			return false, stack.StatusReason
		}
	}
}
