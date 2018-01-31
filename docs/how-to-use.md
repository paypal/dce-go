## How to run dce-go with Mesos Frameworks

In this document, we will cover using dce-go with mesos frameworks such as aurora and marathon. [Vagrantbox setup](https://github.com/paypal/dce-go/blob/master/docs/how-to-use.md) installs and configures  mesos, marathon, apache aurora, dce-go etc.

### Running dce-go on Aurora
#### Configuring Aurora Scheduler
In order to use dce-go with Aurora, we must provide the scheduler a configuration file that contains information on how to run the executor. 
A sample config file for the docker-compose executor is shown below:
```
[
  {
    "executor": {
      "command": {
        "value": "./executor",
        "shell": "true",
        "uris": [
          {
            "cache": false,
            "executable": true,
            "extract": false,
            "value": "http://uri/executor"
          }
        ]
      },
      "name": "docker-compose-executor",
      "resources": [
        {
          "name": "cpus",
          "scalar": {
            "value": 0.25
          },
          "type": "SCALAR"
        },
        {
          "name": "mem",
          "scalar": {
            "value": 256
          },
          "type": "SCALAR"
        }
      ]
    },
    "task_prefix": "compose-"
  }
]
```
The example configuration is also available at dce-go/examples/vagrant/config/docker-compose-executor.json. Multiple executors can be provided in the configuration file. Vagrant setup takes care of setting up aurora scheduler configuration.
More information on how an executor can be configured for consumption by Aurora can be found [here](https://github.com/apache/aurora/blob/master/docs/operations/configuration.md#custom-executors)
under the custom executors section.


[Gorealis](https://github.com/paypal/gorealis) is a go-library for communicating with apache aurora. The [Sample client](https://github.com/paypal/dce-go/blob/opensource/examples/client.go) in this project leverages Gorealis to launch job using aurora scheduler. This is a quick way to try out aurora scheduler apis using our vagrantbox setup. Follow [Installing environment](environment.md) for vagrant setup.
 
**create** command launches a new aurora job. As indicated below, Environment, Role, and Name serve as JobKey, ExecutorName refers to the executor (as specified in scheduler configuration file) to be launched. Further, resources URIs are also specified for the job.

```
case "create":
		fmt.Println("Creating job")
		resp, err := r.CreateJob(job)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(resp.String())

		if resp.ResponseCode == aurora.ResponseCode_OK {
		// 500 will be the timeout for creating job
			if ok, err := monitor.Instances(job.JobKey(), job.GetInstanceCount(), 5, 500); !ok || err != nil {
				_, err := r.KillJob(job.JobKey())
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
```

```
job = realis.NewJob().
			Environment("prod").
			Role("vagrant").
			Name("sampleapp").
			ExecutorName("docker-compose-executor").
			ExecutorData("{}").
			CPU(0.5).
			RAM(64).
			Disk(100).
			IsService(true).
			InstanceCount(1).
			AddPorts(3).
			AddLabel("fileName", "sampleapp/docker-compose.yml,sampleapp/docker-compose-healthcheck.yml").
			AddURIs(true, true, "http://192.168.33.8/app.tar.gz")
```

##### Creating a Job
```
$ cd $GOPATH/src/github.com/paypal/dce-go/examples 
$ go run client.go -executor=compose -url=http://192.168.33.8:8081 -cmd=create
```
Mesos UI would show task details for the launched Job. App can be reached at https://192.168.33.8:${port}/index.html.

File docker-infra-container.yml-generated.yml (shown below) in mesos sandbox for the task has information on exposed ports.
We require to use host port-mapping for 443 -- ''31716 (in this case as indicated)'' in the app url:
(https://192.168.33.8:31716/index.html) 
```
networks:
  default:
    driver: bridge
services:
  networkproxy:
    cgroup_parent: /mesos/011a235c-d49e-4085-9f0e-3e7862d24066
    container_name: vagrant-prod-sampleapp-0-a0694c60-85f5-48c7-90cb-1873dfb05004_networkproxy_0.1
    image: dcego/networkproxy:1.2
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-a0694c60-85f5-48c7-90cb-1873dfb05004
      taskId: vagrant-prod-sampleapp-0-a0694c60-85f5-48c7-90cb-1873dfb05004
    networks:
    - default
    ports:
    - 31282:8081
    - 31645:80
    - 31716:443
version: "2.1"
```

##### Kill a Job
```
$ cd $GOPATH/src/github.com/paypal/dce-go/examples 
$ go run client.go -executor=compose -url=http://192.168.33.8:8081 -cmd=kill
```


### Running dce-go on Marathon

Marathon provides both REST APIs and UI to manage Jobs. 
An example of payload to create applications is provided as follows:
```
{
    "id": "docker-compose-demo",
    "cmd": " ",
    "cpus": 0.5,
    "mem": 64.0,
    "ports":[0,0,0],
    "instances": 1,
    "executor":"./executor",
    "labels": {
        "fileName": "sampleapp/docker-compose.yml,sampleapp/docker-compose-healthcheck.yml"
    },
    "uris":["http://192.168.33.8/app.tar.gz","http://192.168.33.8/example/config.yaml","http://192.168.33.8/general.yaml","http://192.168.33.8/executor"]
}
```

