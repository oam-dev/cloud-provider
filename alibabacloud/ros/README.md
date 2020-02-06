# ROS OAM Framework

## Install ROS Controller with [Helm v3](https://github.com/helm/helm/releases)

```
helm install ros ./charts/ros --set accessKey=<AccessKeyId>,secretKey=<AccessKeySecret>
```

## Apply workloads

```shell script
kubectl apply -f workloads/
```

## Check which workload could be used

Check the workload list you have:

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

Check the detail of one workload

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

## Play With ROS Controller

There is a demo OAM ApplicationConfiguration(AC) and ComponentSchematic YAML file in ${ProjectRootDir}/example/poc/nas\_appconf.yaml

1. run ROS controller
```
go run cmd/ros/main.go
```

2. apply demo
ROS controller will convert appplication configuration to ROS template and the run as a stack.
```
kubectl apply -f example/sls
```

3. delete demo
ROS controller will delete stack which refers to application configuration.
```
kubectl delete applicationconfigurations.core.oam.dev sls-demo
```

## Contributing ROS Controller

### How ROS Controller work

When user update AC content, kubernetes will emit AC change event, ROS Controller use oam-go-sdk to listen and process AC change events , please [see details](https://github.com/oam-dev/oam-go-sdk) to know more.

## Generate workloads
```plain
$ go build gen.go
$ ./gen -i <AccessKeyId> -s <AccessKeySecret>
```
And workloads will be generated to current 'workloads' path.