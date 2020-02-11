package handlers

import (
	"errors"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/v1alpha1"
	"time"

	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/application"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	"github.com/oam-dev/oam-go-sdk/pkg/finalizer"
	"github.com/oam-dev/oam-go-sdk/pkg/oam"
	"k8s.io/apimachinery/pkg/runtime"
)

//TDOO use roscrd in ROSStack case
type AppConfHandler struct {
	OamCrdClient *versioned.Clientset
	RosCrdClient *roscrd.Clientset
	Name         string
}

func (a *AppConfHandler) Handle(ctx *oam.ActionContext, ac runtime.Object, eType oam.EType) error {
	appConf, ok := rosv1alpha1.NewApplicationConfiguration(ac)
	if !ok {
		return errors.New("type mismatch")
	}

	if eType == oam.CreateOrUpdate {
		return a.CreateOrUpdate(ctx, appConf)
	} else if eType == oam.Delete {
		return a.Delete(ctx, appConf)
	}

	return nil
}

func (a *AppConfHandler) Id() string {
	return "appConfHandler"
}

func (a *AppConfHandler) CreateOrUpdate(ctx *oam.ActionContext, appConf *rosv1alpha1.ApplicationConfiguration) (err error) {
	logging.Default.Info("Handle create or update", "AppConf", appConf)

	// ros context
	rosContext, err := ros.GetContext(appConf, a.OamCrdClient, a.RosCrdClient)
	if err != nil {
		return
	}

	appName := appConf.Name

	// check progressing
	appStackData, err := application.GetAppStackData(rosContext)
	if err != nil {
		return
	}
	if application.IsAppStackProgressing(appStackData) {
		logging.Default.Info("Application is still progressing", application.AppName, appName)
		return
	}

	// template
	logging.Default.Info("Generating ROS template for application", application.AppName, appName)
	template, err := ros.NewTemplate(rosContext, appConf)
	if err != nil {
		logging.Default.Error(err, "Generate ROS template for application failed", application.AppName, appName)
		err = application.SetAppStackError(rosContext, err)
		return
	}
	logging.Default.Info("Generate ROS template for application successfully", application.AppName, appName)
	//bytes, _ := json.MarshalIndent(template, "", "  ")
	//fmt.Println("Template:", string(bytes))

	// check template same
	templateBodyByte, _ := template.Marshal()
	templateBody := string(templateBodyByte)
	if !application.IsAppStackFailed(appStackData) && appStackData[application.TemplateBody] == templateBody {
		logging.Default.Info("Application stack template is completely same", application.AppName, appName)
		return
	}

	// get stack
	stackName := appName
	stack, err := application.GetStackFromAppStack(rosContext)
	if err != nil {
		return err
	}

	// add cleanup
	err = addCleanUpFinalizer(rosContext)
	if err != nil {
		return err
	}

	if stack == nil {
		// create stack
		logging.Default.Info("Creating ROS stack", ros.StackName, stackName)
		stack, err = ros.NewStack(rosContext, stackName, template)
		if err != nil {
			err = application.SetAppStackError(rosContext, err)
			return err
		}
		err = application.SetAppStackIdAndTemplate(rosContext, stack.Id, templateBody)
		if err != nil {
			return err
		}
		err = application.SetAppStackProgressing(rosContext)
	} else {
		// update stack
		logging.Default.Info("Updating ROS stack", ros.StackName, stackName, ros.StackId, stack.Id)
		err = stack.Update(rosContext, template)
		if err != nil {
			if ros.IsStackSame(err) {
				logging.Default.Info("Stack is completely same")
				return nil
			} else if ros.IsStackNotFound(err) {
				// create stack
				logging.Default.Info("Stack not exist. Creating ROS stack", ros.StackName, stackName)
				stack, err = ros.NewStack(rosContext, stackName, template)
				if err != nil {
					err = application.SetAppStackError(rosContext, err)
					return err
				}
				err = application.SetAppStackIdAndTemplate(rosContext, stack.Id, templateBody)
				if err != nil {
					return err
				}
			} else {
				err = application.SetAppStackError(rosContext, err)
				return err
			}
		}
		err = application.SetAppStackProgressing(rosContext)
	}

	go waitStackDoneAndSaveOutputs(rosContext, stack, false)

	return
}

func (a *AppConfHandler) Delete(ctx *oam.ActionContext, appConf *rosv1alpha1.ApplicationConfiguration) (err error) {
	logging.Default.Info("Handle delete", "AppConf", appConf)

	appName := appConf.Name

	// ros context
	rosContext, err := ros.GetContext(appConf, a.OamCrdClient, a.RosCrdClient)
	if err != nil {
		return
	}

	// check progressing
	data, err := application.GetAppStackData(rosContext)
	if err != nil {
		return
	}
	if application.IsAppStackProgressing(data) {
		logging.Default.Info("Application is still progressing", application.AppName, appName)
		waitAppDone(rosContext)
	}

	// stack
	stack, err := application.GetStackFromAppStack(rosContext)
	if err != nil {
		return
	}

	if stack == nil {
		logging.Default.Info("No need to delete stack. There is no stack for application", application.AppName, appName)
		err = removeCleanUpFinalizer(rosContext)
		return err
	}

	logging.Default.Info("Deleting ROS stack", ros.StackName, stack.Name, ros.StackId, stack.Id)
	err = stack.Delete(rosContext)
	if err != nil {
		if ros.IsStackNotFound(err) {
			err = application.DeleteAppStack(rosContext)
			if err != nil {
				return err
			}
			err = removeCleanUpFinalizer(rosContext)
			return err
		} else {
			err = application.SetAppStackError(rosContext, err)
			return err
		}
	}

	go waitStackDoneAndSaveOutputs(rosContext, stack, true)

	return err
}

func addCleanUpFinalizer(rosContext *ros.Context) (err error) {
	logging.Default.Info("Add ROS finalizer")
	appConf, err := rosContext.GetAppConf()
	if err != nil {
		return err
	}

	finalizer.Add(appConf.ToObject(), config.ROS_FINALIZER)

	err = rosContext.UpdateAppConf(appConf)
	return err
}

func removeCleanUpFinalizer(rosContext *ros.Context) (err error) {
	logging.Default.Info("Remove ROS finalizer")
	appConf, err := rosContext.GetAppConf()
	if err != nil {
		return err
	}

	finalizer.Remove(appConf.ToObject(), config.ROS_FINALIZER)

	err = rosContext.UpdateAppConf(appConf)
	return err
}

func waitAppDone(rosContext *ros.Context) {
	for {
		time.Sleep(5 * time.Second)
		logging.Default.Info("Waiting app done", "AppName", rosContext.AppConf.Name)
		status, err := application.GetAppStackStatus(rosContext)
		if err != nil {
			logging.Default.Error(err, "Waiting app done error")
		}
		if status == application.Ready || status == application.Failed {
			logging.Default.Info("App done", "AppName", rosContext.AppConf.Name, "AppStatus", status)
			return
		}
	}
}

func waitStackDoneAndSaveOutputs(rosContext *ros.Context, stack *ros.Stack, deleteAppStack bool) {
	var err error
	appName := rosContext.AppConf.Name
	success, statusReason := stack.WaitUntilDone(rosContext)

	if success {
		logging.Default.Info("Stack runs done")

		if !deleteAppStack {
			err = application.SetAppStackReady(rosContext)
			if err != nil {
				logging.Default.Error(err, "Set app stack ready failed", "AppName", appName)
			}
		}

		switch ros.StackStatusType(stack.Status) {
		case ros.CreateComplete:
			fallthrough
		case ros.UpdateComplete:
			fallthrough
		case ros.CheckComplete:
			application.SaveAppStackOutputs(rosContext, stack)
		}
	} else {
		logging.Default.Error(err, "Stack runs failed")
		err = application.SetAppStackError(rosContext, errors.New(statusReason))
		if err != nil {
			logging.Default.Error(err, "Set app stack error failed", "AppName", appName)
		}
	}

	if deleteAppStack {
		err = application.DeleteAppStack(rosContext)
		if err != nil {
			logging.Default.Error(err, "Delete app stack error failed", "AppName", appName)
		}

		err = removeCleanUpFinalizer(rosContext)
		if err != nil {
			logging.Default.Error(err, "Remove CleanUp finalizer failed", "AppName", appName)
		}
	}
	return
}
