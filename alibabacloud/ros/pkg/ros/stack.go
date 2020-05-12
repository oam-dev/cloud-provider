package ros

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"time"
)

const DryRunFakeStack = "DryRunFakeStack"

const (
	StackId     = "StackId"
	StackName   = "StackName"
	StackStatus = "StackStatus"
	OutputKey   = "OutputKey"
	OutputValue = "OutputValue"
)

type Stack struct {
	Client        *rosapi.Client
	Id            string                   `json:"Id"`
	Name          string                   `json:"Name"`
	Status        string                   `json:"Status"`
	StatusReason  string                   `json:"StatusReason"`
	Outputs       []map[string]interface{} `json:"Outputs"`
	dryRunHandler func(stack *Stack, request requests.AcsRequest) error
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

// stackOption defines secret option
type stackOption struct {
	DryRunHandler func(stack *Stack, request requests.AcsRequest) error
}

// StackOption has methods to work with secret option.
type StackOption interface {
	apply(*stackOption)
}

// stackFuncOption defines function used for secret option
type stackFuncOption struct {
	f func(*stackOption)
}

// apply executes stackFuncOption's func
func (fdo *stackFuncOption) apply(do *stackOption) {
	fdo.f(do)
}

// newStackFuncOption returns function option
func newStackFuncOption(f func(*stackOption)) *stackFuncOption {
	return &stackFuncOption{
		f: f,
	}
}

// WithDryRunHandler sets client set in secret option
func WithDryRunHandler(dryRunHandler func(stack *Stack, request requests.AcsRequest) error) StackOption {
	return newStackFuncOption(func(o *stackOption) {
		o.DryRunHandler = dryRunHandler
	})
}

// NewStack parses application configuration and creates a ROS stack
func NewStack(appContext *appconf.Context, stackName string, template *Template, opts ...StackOption) (stack *Stack, err error) {
	// init option
	o := &stackOption{}
	for _, opt := range opts {
		opt.apply(o)
	}

	if o.DryRunHandler == nil {
		o.DryRunHandler = func(stack *Stack, request requests.AcsRequest) error {
			logging.Default.Info("Dry run", "request", request)
			return nil
		}
	}

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

	stack = &Stack{
		Client:        appContext.RosClient,
		Name:          stackName,
		dryRunHandler: o.DryRunHandler,
	}

	if appContext.DryRun {
		stack.Id = DryRunFakeStack
		return stack, stack.dryRunHandler(stack, request)
	}

	response, err := appContext.RosClient.CreateStack(request)
	if err != nil {
		return
	}

	stack.Id = response.StackId
	return
}

func (s *Stack) Update(template *Template) error {
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
	request.StackId = s.Id
	request.Parameters = &parameters
	request.TemplateBody = string(templateBody)

	if s.Id == DryRunFakeStack {
		return s.dryRunHandler(s, request)
	}

	_, err = s.Client.UpdateStack(request)
	if err != nil {
		return err
	}

	return nil
}

func (s *Stack) Delete() error {
	request := rosapi.CreateDeleteStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.StackId = s.Id

	if s.Id == DryRunFakeStack {
		return s.dryRunHandler(s, request)
	}

	_, err := s.Client.DeleteStack(request)
	if err != nil {
		return err
	}

	return nil
}

func (s *Stack) Refresh() error {
	request := rosapi.CreateGetStackRequest()
	request.AppendUserAgent("Service", config.RosCtrlConf.UserAgent)
	request.StackId = s.Id

	if s.Id == DryRunFakeStack {
		return s.dryRunHandler(s, request)
	}

	resp, err := s.Client.GetStack(request)
	if err != nil {
		return err
	}

	s.Name = resp.StackName
	s.Status = resp.Status
	s.StatusReason = resp.StatusReason
	s.Outputs = resp.Outputs

	return nil
}

func (s *Stack) IsInDeleteStatus() bool {
	status := StackStatusType(s.Status)
	return status == DeleteInProgress || status == DeleteFailed || status == DeleteComplete
}

func (s *Stack) WaitUntilDone() (success bool, statusReason string) {
	for {
		time.Sleep(time.Duration(config.RosCtrlConf.StackCheckInterval) * time.Second)
		err := s.Refresh()
		if err != nil {
			logging.Default.Error(err, "Refresh s error", StackId, s.Id, StackName, s.Name)
			continue
		}

		logging.Default.Info("Stack info", StackId, s.Id, StackName, s.Name, StackStatus, s.Status)

		switch StackStatusType(s.Status) {
		// success
		case CreateComplete:
			fallthrough
		case UpdateComplete:
			fallthrough
		case DeleteComplete:
			fallthrough
		case CheckComplete:
			logging.Default.Info("Stack check done", StackId, s.Id, StackName, s.Name, StackStatus, s.Status)
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
			logging.Default.Info("Stack check failed", StackId, s.Id, StackName, s.Name, StackStatus, s.Status)
			return false, s.StatusReason
		}
	}
}
