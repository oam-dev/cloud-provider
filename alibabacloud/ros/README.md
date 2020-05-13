# ROS OAM Controller

## Introduction

`ROS OAM Controller` is an implementation of Alibaba Cloud resource orchestration
that follows OAM standards. It is based on [ROS](https://www.alibabacloud.com/help/doc-detail/28852.html),
and you can easily orchestrate various service resources of Alibaba Cloud through OAM spec.

## Installation & Running

### By using [Helm v3](https://github.com/helm/helm/releases)

```shell script
helm install ros ./charts/ros --set accessKey=<AccessKeyId>,secretKey=<AccessKeySecret>
```

### Or by running go run

```shell script
go run cmd/ros/main.go --access-key-id=<AccessKeyId> --access-key-secret=<AccessKeySecret>
```

## Quick Start
> Please make sure you have a kubernetes cluster running.

After installation and running, let's start with an example which creates 
Alibaba Cloud [SLS](https://www.alibabacloud.com/help/doc-detail/48869.htm) project, logstore and index.

### Write OAM Configurations
In `example/sls`, there are server yaml files which follow OAM standards:
- `appconf_sls.yaml` is an OAM application which specify three SLS components
- `comp_sls_project.yaml` is an OAM component which indicates Alibaba Cloud SLS project
- `comp_sls_logstore.yaml` is an OAM component which indicates Alibaba Cloud SLS logstore
- `comp_sls_index.yaml` is an OAM component which indicates Alibaba Cloud SLS index

These files will convert to a ROS template by this controller and create
the Alibaba Cloud resources you want.

### Create Resources by OAM Configurations
By applying OAM configurations, you can create SLS resources.

```shell script
kubectl apply -f example/sls
```

After a few seconds visit the [ROS Console](https://rosnext.console.aliyun.com/cn-hangzhou/stacks),
and you will see the created stack, which contains related SLS resources.

### Delete Resources from OAM Configurations

By deleting OAM configurations files, you can delete SLS resources.

```shell script
kubectl delete applicationconfigurations.core.oam.dev sls-demo
```

After a few seconds visit the [ROS Console](https://rosnext.console.aliyun.com/cn-hangzhou/stacks),
and you will see the stack and related SLS resources are deleted.


## Usage
### Application Cmdline
The ROS OAM Controller application supports many options to run with:
```
Usage of ./main:
  -access-key-id string
    	User's access key ID.
  -access-key-secret string
    	User's Access key secret.
  -credential-secret-name string
    	User's credential secret name.
  -endpoint string
    	ROS api endpoint. (default "https://ros.aliyuncs.com")
  -env string
    	App running environment. (default "test")
  -kubeconfig string
    	Paths to a kubeconfig. Only required if out-of-cluster.
  -leader-election-namespace string
    	Leader election namespace. (default "default")
  -master --kubeconfig
    	(Deprecated: switch to --kubeconfig) The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
  -metrics-addr string
    	The address the metric endpoint binds to. (default ":8080")
  -namespace string
    	App namespace. (default "default")
  -region-id string
    	Region where ROS creates resources from. (default "cn-hangzhou")
  -ros-crd
    	Whether this controller work as ROS or OAM CRD.
  -service-user-agent string
    	Current service/application name which will be set to User-Agent for identification.
  -update-app
    	Whether update application status.
```

You can specify one or many of them to run the application.

### Workloads
- Apply workloads
```shell script
kubectl apply -f workloads/
```

- Check which workload could be used

```shell script
$ kubectl get workloadtypes
NAME                                      AGE
actiontrail-trail                         53s
actiontrail-traillogging                  53s
alibaba-service                           18m
apigateway-api                            53s
apigateway-app                            53s
apigateway-authorization                  53s
...
```

- Check the detail of one workload

```shell script
$ kubectl get workloadtypes actiontrail-trail -o yaml
apiVersion: core.oam.dev/v1alpha1
kind: WorkloadType
metadata:
  name: actiontrail-trail
  namespace: default
spec:
  group: ros.aliyun.com
  names:
    kind: ACTIONTRAIL_Trail
  version: v1alpha1
  workloadSettings: |-
    {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "type": "object",
      "required": [
        "OssBucketName",
        "RoleName",
        "Name"
      ],
      "properties": {
        "EventRW": {
          "type": "string",
          "description": "Indicates whether the event is a read or a write event. Valid values: Read, Write, and All. Default value: Write.",
          "default": "Write",
          "Enum": [
            "All",
            "Read",
            "Write"
          ]
        },
        "Name": {
          "type": "string",
          "description": "The name of the trail to be created, which must be unique for an account."
        },
        "OssBucketName": {
          "type": "string",
          "description": "The OSS bucket to which the trail delivers logs. Ensure that this is an existing OSS bucket."
        },
        "OssKeyPrefix": {
          "type": "string",
          "description": "The prefix of the specified OSS bucket name. This parameter can be left empty."
        },
        "RoleName": {
          "type": "string",
          "description": "The RAM role in ActionTrail permitted by the user."
        },
        "SlsProjectArn": {
          "type": "string",
          "description": "The unique ARN of the Log Service project."
        },
        "SlsWriteRoleArn": {
          "type": "string",
          "description": "The unique ARN of the Log Service role."
        }
      }
    }
```

- Sync workloads will fetch all resource info and generate workloads to current `workloads` path.

```shell script
go run gen.go -i <AccessKeyId> -s <AccessKeySecret>
```

- You can apply them again to update this info in cluster.
```shell script
kubectl apply -f workloads/
```

## Contributing ROS Controller

### How ROS Controller work

When user update AC content, kubernetes will emit AC change event, ROS Controller use oam-go-sdk to listen and process
AC change events , please [see details](https://github.com/oam-dev/oam-go-sdk) to know more.
