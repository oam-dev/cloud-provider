package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appstack"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	"github.com/oam-dev/oam-go-sdk/pkg/finalizer"
	"github.com/oam-dev/oam-go-sdk/pkg/oam"
	ks8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type AppConfHandler struct {
	OamCrdClient *versioned.Clientset
	RosCrdClient *roscrd.Clientset
	Name         string
}

func (a *AppConfHandler) Handle(ctx *oam.ActionContext, ac runtime.Object, eType oam.EType) error {
	appConf, ok := appconf.NewAppConf(ac)
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

func (a *AppConfHandler) CreateOrUpdate(ctx *oam.ActionContext, appConf *appconf.AppConf) (err error) {
	appConfStr, err := json.MarshalIndent(appConf, "", "  ")
	if err != nil {
		return err
	}
	logging.Default.Info(fmt.Sprintf("Handle create or update appConf: \n%s", string(appConfStr)))

	// ros context
	appContext, err := appconf.NewContext(appConf, a.OamCrdClient, a.RosCrdClient)
	if err != nil {
		return
	}

	// app stack
	appStack := appstack.NewAppStack(appContext)
	appStackName := appStack.GetName()

	// check progressing
	isProgressing, err := appStack.IsProgressing()
	if err != nil {
		return
	}
	if isProgressing {
		logging.Default.Info("Application is still progressing", appstack.AppStackName, appStackName)
		return
	}

	// template
	logging.Default.Info("Generating ROS template for application", appstack.AppStackName, appStackName)
	template, err := ros.NewTemplate(appContext, appConf)
	if err != nil {
		logging.Default.Error(err, "Generate ROS template for application failed", appstack.AppStackName, appStackName)
		err = appStack.SetError(err)
		return
	}
	logging.Default.Info("Generate ROS template for application successfully", appstack.AppStackName, appStackName)

	// check template same
	templateBodyByte, _ := template.Marshal()
	templateBody := string(templateBodyByte)
	isFailed, err := appStack.IsFailed()
	if err != nil {
		return
	}
	appStackData, err := appStack.GetData()
	if err != nil {
		return
	}
	if !isFailed && appStackData[appstack.TemplateBody] == templateBody {
		logging.Default.Info("Application stack template is completely same", appstack.AppStackName, appStackName)
		return
	}

	// get stack
	stackName := appStackName
	stack, err := appStack.GetStack()
	if err != nil {
		return err
	}

	// add cleanup
	err = addCleanUpFinalizer(appContext)
	if err != nil {
		return err
	}

	if stack == nil {
		// create stack
		logging.Default.Info("Creating ROS stack", ros.StackName, stackName)
		stack, err = ros.NewStack(appContext, stackName, template)
		if err != nil {
			err = appStack.SetError(err)
			return err
		}
		err = appStack.SetIdAndTemplate(stack.Id, templateBody)
		if err != nil {
			return err
		}
		err = appStack.SetProgressing()
	} else {
		// update stack
		logging.Default.Info("Updating ROS stack", ros.StackName, stackName, ros.StackId, stack.Id)
		err = stack.Update(template)
		if err != nil {
			if ros.IsStackSame(err) {
				logging.Default.Info("Stack is completely same")
				return nil
			} else if ros.IsStackNotFound(err) {
				// create stack
				logging.Default.Info("Stack not exist. Creating ROS stack", ros.StackName, stackName)
				stack, err = ros.NewStack(appContext, stackName, template)
				if err != nil {
					err = appStack.SetError(err)
					return err
				}
				err = appStack.SetIdAndTemplate(stack.Id, templateBody)
				if err != nil {
					return err
				}
			} else {
				err = appStack.SetError(err)
				return err
			}
		} else {
			err = appStack.SetIdAndTemplate(stack.Id, templateBody)
			if err != nil {
				return err
			}
		}
		err = appStack.SetProgressing()
	}

	go waitStackDoneAndSaveOutputs(appContext, appStack, stack)

	return
}

func (a *AppConfHandler) Delete(ctx *oam.ActionContext, appConf *appconf.AppConf) (err error) {
	appConfStr, err := json.MarshalIndent(appConf, "", "  ")
	if err != nil {
		return err
	}
	logging.Default.Info(fmt.Sprintf("Handle delete AppConf: \n%s", string(appConfStr)))

	// ros context
	appContext, err := appconf.NewContext(appConf, a.OamCrdClient, a.RosCrdClient)
	if err != nil {
		return
	}

	// app stack
	appStack := appstack.NewAppStack(appContext)
	appStackName := appStack.GetName()

	// check progressing
	isProgressing, err := appStack.IsProgressing()
	if err != nil {
		return
	}
	if isProgressing {
		logging.Default.Info("Application is still progressing", appstack.AppStackName, appStackName)
		status := appStack.WaitUntilDone()
		if status == appstack.Deleted {
			return nil
		}
	}

	// stack
	stack, err := appStack.GetStack()
	if err != nil {
		return
	}

	if stack == nil {
		logging.Default.Info("No need to delete stack. There is no stack for application", appstack.AppStackName, appStackName)
		err = removeCleanUpFinalizer(appContext)
		return err
	}

	logging.Default.Info("Deleting ROS stack",
		ros.StackName, stack.Name,
		ros.StackId, stack.Id,
		appstack.AppStackName, appStackName)
	err = stack.Delete()
	if err != nil {
		if ros.IsStackNotFound(err) {
			err = appStack.Delete()
			if err != nil {
				return err
			}
			err = removeCleanUpFinalizer(appContext)
			return err
		} else {
			err = appStack.SetError(err)
			return err
		}
	} else {
		err := appStack.SetProgressing()
		if err != nil {
			return err
		}
	}

	go waitStackDoneAndSaveOutputs(appContext, appStack, stack)

	return err
}

func RecoverProgressingAppStacks(oamCrdClient *versioned.Clientset, rosCrdClient *roscrd.Clientset) {
	logging.Default.Info("Load progressing app stacks")
	appStacks, err := appstack.LoadProgressingAppStacks(oamCrdClient, rosCrdClient)
	if err != nil {
		logging.Default.Error(err, "Load progressing app stacks error")
		return
	}
	for _, appStack := range appStacks {
		appStackName := appStack.GetSecretName()
		stack, err := appStack.GetStack()
		if err != nil {
			logging.Default.Error(err, "Get stack from app stack failed", appstack.AppStackName, appStackName)
			continue
		}

		logging.Default.Info("Recover app stack", appstack.AppStackName, appStackName)
		go waitStackDoneAndSaveOutputs(appStack.GetContext(), appStack, stack)
	}
}

func addCleanUpFinalizer(appContext *appconf.Context) (err error) {
	logging.Default.Info("Add ROS finalizer")
	appConf, err := appconf.GetAppConfFromContext(appContext)
	if err != nil {
		return err
	}

	finalizer.Add(appConf.ToObject(), config.ROS_FINALIZER)

	err = appConf.Update(appContext, appConf)
	return err
}

func removeCleanUpFinalizer(appContext *appconf.Context) (err error) {
	logging.Default.Info("Remove ROS finalizer")
	appConf, err := appconf.GetAppConfFromContext(appContext)
	if err != nil {
		if ks8errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	finalizer.Remove(appConf.ToObject(), config.ROS_FINALIZER)

	err = appConf.Update(appContext, appConf)
	return err
}

func waitStackDoneAndSaveOutputs(appContext *appconf.Context, appStack appstack.AppStackInterface, stack *ros.Stack) {
	var err error
	AppStackName := appStack.GetName()
	success, statusReason := stack.WaitUntilDone()
	deleteAppStack := stack.IsInDeleteStatus()

	if success {
		logging.Default.Info("Stack runs done")

		switch ros.StackStatusType(stack.Status) {
		case ros.CreateComplete:
			fallthrough
		case ros.UpdateComplete:
			fallthrough
		case ros.CheckComplete:
			appStack.SaveOutputs(stack)
		}

		if !deleteAppStack {
			err = appStack.SetReady()
			if err != nil {
				logging.Default.Error(err, "Set app stack ready failed", appstack.AppStackName, AppStackName)
			}
		}
	} else {
		logging.Default.Error(err, "Stack runs failed")
		err = appStack.SetError(errors.New(statusReason))
		if err != nil {
			logging.Default.Error(err, "Set app stack error failed", appstack.AppStackName, AppStackName)
		}
	}

	if deleteAppStack {
		err = appStack.Delete()
		if err != nil {
			logging.Default.Error(err, "Delete app stack error failed", appstack.AppStackName, AppStackName)
		}

		err = removeCleanUpFinalizer(appContext)
		if err != nil {
			logging.Default.Error(err, "Remove CleanUp finalizer failed", appstack.AppStackName, AppStackName)
		}
	}
	return
}
