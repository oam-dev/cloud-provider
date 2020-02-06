package logging

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var SetUp = ctrl.Log.WithName("setup")
var Default = ctrl.Log.WithName("default")

func Init() {
	if config.RosCtrlConf.LogToFile {
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename: config.RosCtrlConf.LogFilePath,
			MaxSize:  200, // megabytes
			MaxAge:   5,   // days
		})
		ctrl.SetLogger(zap.New(zap.UseDevMode(config.RosCtrlConf.LoggerDebug), zap.WriteTo(w)))
	} else {
		ctrl.SetLogger(zap.New(zap.UseDevMode(config.RosCtrlConf.LoggerDebug)))
	}
}
