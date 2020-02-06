module github.com/oam-dev/cloud-provider/alibabacloud/ros

go 1.12

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.293
	github.com/oam-dev/oam-go-sdk v0.0.0-20200116043142-d934017ed4cd
	github.com/urfave/cli/v2 v2.0.0
	go.uber.org/zap v1.9.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
