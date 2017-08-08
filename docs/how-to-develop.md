## How to develop dce-go

In this document, we will cover implementation details of dce-go. As mentioned before, running multiple pods on the same host may create many conflicts (containerId's , ports etc.). Executor takes care of resolving these conflicts by generating a new compose file. New compose file generation is powered via plugin mechanism. General plugin modifies compose sections so as to avoid conflict among running pods unique for a host and supports basic requirements for running a pod. Appropriate custom logic can be implemented via a plugin and injected to dce-go. The plugin mechanism will allow developers to extend and modify default behavior via set of static plugins. Main configuration file maintains plugin order. They are invoked in that order at appropriate executor callback interface.

dce-go implements [mesos executor callback interface](https://github.com/mesos/mesos-go/blob/master/api/v0/executor/exectype.go). 
LaunchTask, KillTask and Shutdown methods are of our interest where plugins are invoked. LaunchTask is invoked when a task is launched on executor (initiated via SchedulerDriver.LaunchTasks), while KillTask is invoked when a task running within this executor is killed via SchedulerDriver.KillTask and Shutdown is invoked via SchedulerDriver.Shutdown. This document covers these scenarios in detail below.


##### State Diagram: Executor LaunchTask

LaunchTask is executor callback for launching task on this executor. PreLaunch and PostLaunch plugin methods are invoked during this callback. Below diagram describes steps involved for LaunchTask.

<p align="center">
  <img src="https://github.com/paypal/dce-go/blob/master/docs/images/launchtask.png?raw=true" alt="LaunchTask"/>
</p>

Pre Launch Task Plugin: Static plugins allow to inject custom behavior. For instance, General plugin via pre-launch modify/inject sections of compose such as labels, env, cgroups, networks etc.

Pull Images: Pulls docker images after PreLaunchTask Plugins are executed in order.

Compose up:  Launches pod based on plugin generated compose files.

Post Launch Task Plugin: Likewise, post-launch aims at injecting custom logic after pod is launched.

Pod Monitor: Once task is running, executor launches pod monitor to periodically (configurable value) monitor pod status. If container(s) become unhealthy, then it brings down the pod and updates task status to mesos. It also stops on receiving KillTask.

##### State Diagram: Executor KillTask

<p align="center">
  <img src="https://github.com/paypal/dce-go/blob/master/docs/images/deletetask.png?raw=true" alt="KillTask"/>
</p>

Scenarios of Kill Task

1. "pod monitor" state: KillTask is invoked via mesos callback after task is running and monitored. Executor will  cleanup pod as well as update task state as KILLED.
2. "failed" state: Pod Monitor detects unhealthy state and triggers pod cleanup and task update and at the same time KillTask is invoked by mesos while pod is in program of shutdown. Here, KillTask will be no-op.
3. "killed" state: Pod is already in program of shutdown. Here, KillTask will be no-op.

##### Sequence Diagram: Pod Monitor

<p align="center">
  <img src="https://github.com/paypal/dce-go/blob/master/docs/images/podmonitor.png?raw=true" alt="Pod Monitor"/>
</p>

PodMonitor: Pod Monitor is launched by dce-go once task is running.  Its responsibility is to monitor the health status of pod until pod becomes unhealthy. Additionally, Pod Monitor will trigger cleanup pod and update task state as FAILED once pod is unhealthy. 


#### General plugin

DCE-GO comes with default General Plugin. This Plugin updates compose files so that multiple pods are able to launch on a host. It largely covers following:
* Decorate various compose sections  to resolve all the conflicts.
* Tag each container with specific taskId and executorId. This information is used to clean up pod.
* Adding pod to parent mesos task cgroups.
* Creating infrastructure container for allowing to collapse network namespace for containers in a pod.


#### Plugin Development
Any additional custom logic can be supported via a plugin implementation. It requires implementing ComposePlugin interface and registering as a plugin. Details below.


##### Implementing  ComposePlugin interface
```
type ComposePlugin interface {
       // PreLaunchTask is invoked prior to pod launch in  context of executor LaunchTask callback.
       PreLaunchTask(ctx *context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error
       
       // PostLaunchTask is invoked after pod launch in context of executor LaunchTask callback.
       PostLaunchTask(ctx *context.Context, composeFiles []string, taskInfo *mesos.TaskInfo) (string, error)
       
       // PreKillTask is invoked prior to killing pod in context of executor KillTask callback. 
       PreKillTask(taskInfo *mesos.TaskInfo) error
       
       // PostKillTask is invoked after killing pod in context of executor KillTask callback. 
       PostKillTask(taskInfo *mesos.TaskInfo) error
       
       // Shutdown is invoked prior to executor shutdown in context of Shutdown callback. 
       Shutdown(executor.ExecutorDriver) error
}
```
PreLaunchTask and PostLaunchTask have Context object as first parameter. This is used to pass around parsed compose files so as to avoid loading from files by individual plugins.
Below sample code illustrates a sample plugin loading parsed compose file from context in an object of ServiceDetail type.

##### Sample Plugin implementation
Example plugin implementation can be found [here](../plugin/example/impl.go)
```
func (ex *exampleExt) PreLaunchTask(ctx *context.Context, composeFiles *[]string, executorId string, taskInfo *mesos.TaskInfo) error {
	logger.Println("PreLaunchTask Starting")
       // docker compose YML files are saved in context as type SERVICE_DETAIL which is map[interface{}]interface{}.
	// Massage YML files and save it in context.
	// Then pass to next plugin.

	// Get value from context
	filesMap := (*ctx).Value(types.SERVICE_DETAIL).(types.ServiceDetail)

	// Add label in each service, in each compose YML file
	for _, file := range *composeFiles {
		servMap := filesMap[file][types.SERVICES].(map[interface{}]interface{})
		for serviceName := range servMap {
			containerDetails := filesMap[file][types.SERVICES].(map[interface{}]interface{})[serviceName].(map[interface{}]interface{})
			if labels, ok := containerDetails[types.LABELS].(map[interface{}]interface{}); ok {
				labels["com.company.label"] = "awesome"
				containerDetails[types.LABELS] = labels
			}

		}

	}

	// Save the changes back to context
	*ctx = context.WithValue(*ctx, types.SERVICE_DETAIL, filesMap)

	return nil
}

func (ex *exampleExt) PostLaunchTask(ctx *context.Context, composeFiles []string, taskInfo *mesos.TaskInfo) (string, error) {
	logger.Println("PostLaunchTask Starting")
	return "", nil
}

func (ex *exampleExt) PreKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PreKillTask Starting")
	return nil
}

func (ex *exampleExt) PostKillTask(taskInfo *mesos.TaskInfo) error {
	logger.Println("PostKillTask Starting")
	return nil
}

func (ex *exampleExt) Shutdown(executor.ExecutorDriver) error {
	logger.Println("Shutdown Starting")
	return nil
}
```
##### Registering plugins
1. dce-go keeps a plugin registry map to keep track of all registered plugins.  Plugin registration is supported via package init( ) function. Below sample code snippet illustrates plugin registration.  In addition, extra configuration relevant to plugin can be loaded as well. 
    ```
    func init() {
       logger = log.WithFields(log.Fields{
              "plugin": "example",
       })
       log.SetOutput(os.Stdout)

       logger.Println("Plugin Registering")

       plugin.ComposePlugins.Register(new(exampleExt), "example")
       
       //Merge plugin config
       config.ConfigInit(utils.SearchFile(".", "example.yaml"))
    }
    ```
2. Importing your plugin as a package solely for side-effects.
   Open dce/main.go and add following lines in imports.
   ```
      _ "github.com/paypal/dce-go/plugin/<your plugin package name>"
   ```
      Here is an example.
   ```
       _ "github.com/paypal/dce-go/plugin/example"
   ```
3. Last but not least, an order of plugins need to be specified in the main configuration file([config.yaml](../config/config.yaml)). Plugins are automatically invoked in that order at appropriate executor callback interface . 

    Here is an example of config.yaml
    ```
    launchtask:
       podmonitorinterval: 10000
       pullretry: 3
       maxretry: 3
       retryinterval: 10000
       timeout: 500000
    plugins:
       pluginorder: general,example
    cleanpod:
       timeout: 20
       unhealthy: true
       cleanvolumeandcontaineronmesoskill: true
       cleanimageonmesoskill: true
    ```
    In this case, dce will invoke general plugin followed by example plugin.

##### Let's go over an example to see how plugin massage yml files.

docker-compose.yml.
```
version: "2.1"
services:
  web:
    image: dcego/web:1.0
    volumes:
      - "./app:/src/app"
    ports:
      - "8081:8081"
  nginx:
    image: dcego/nginx:1.0
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /www/public
    volumes_from:
      - web
```
After executing general plugin, section names are prefixed to avoid conflicts. Also you'd notice that network namespace for the pod is collapsed to infra container via network_mode section. In the example below, it points to service:networkproxy. Service definition for infra container is covered later in this document. 
```
services:
  nginx:
    cgroup_parent: /mesos/857fd7a0-df05-458b-bd72-f5c280790aa0
    image: dcego/nginx:1.0
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      taskId: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
    network_mode: service:networkproxy
    volumes:
    - /www/public
    volumes_from:
    - web
  web:
    cgroup_parent: /mesos/857fd7a0-df05-458b-bd72-f5c280790aa0
    image: dcego/web:1.0
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      taskId: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
    network_mode: service:networkproxy
    volumes:
    - ./app:/src/app
version: "2.1"
```
Example plugin adds a label "com.company.label' to all services. Likewise, various compose sections can be modified.
```
services:
  nginx:
    cgroup_parent: /mesos/857fd7a0-df05-458b-bd72-f5c280790aa0
    image: dcego/nginx:1.0
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      taskId: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
    network_mode: service:networkproxy
    volumes:
    - /www/public
    volumes_from:
    - web
  web:
    cgroup_parent: /mesos/857fd7a0-df05-458b-bd72-f5c280790aa0
    image: dcego/web:1.0
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      taskId: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      com.company.label: awesome
    network_mode: service:networkproxy
    volumes:
    - ./app:/src/app
version: "2.1"
```

##### Infra container 

Below is the infra container section, added by General Plugin. Pod containers attach to infra container network. It also has port mapping section. In example below, application port 80 is mapped to host port 31695. Infra container information such as image, container, network and driver is captured in General Plugin, discussed later in this document. 

```
networks:
  default:
    driver: <your driver>
services:
  networkproxy:
    cgroup_parent: /mesos/857fd7a0-df05-458b-bd72-f5c280790aa0
    container_name: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887_networkproxy_0.1
    image: dcego/networkproxy:1.2
    labels:
      executorId: compose-vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
      taskId: vagrant-prod-sampleapp-0-ae1549b2-93f9-436f-9070-b387c878b887
    networks:
    - default
    ports:
    - 31516:8081
    - 31695:80
    - 31754:443
version: "2.1"
```

Note that infra container is ONLY used for bridge mode and custom network (if defined in general plugin config). It is not added for network_mode -- "host" and "none". Here are the supported scenarios:
1. Default network is bridge mode in absence of missing networks section. Infra container section is added with default bridge network. Here is a [sample example](../examples/manifest/docker-compose-bridge.yml) along with [generated compose file](../examples/manifest/docker-compose-bridge-generated.yml)

2. If general plugin config (covered later in document) has defines networks settting as pre-existing then Infra container uses this network instead. 

3. If network_mode is set to "host" and "none" then infra container is not used by general plugin. 

For details, please follow [sample compose manifests here](../examples/manifest)

#### How to build DCE-GO

This project has makefile to build and upload binary for vagrant setup. Below following commands helps achieve this.
```
   $ cd $GOPATH/src/github.com/paypal/dce-go
   $ make build
```

To upload binary file to nginx used in vagrant setup. Note that upload target is only meant for vagrant setup. 
```
   $ make upload
```

#### DCE-GO Configuration Files
There are 2 types of configuration files: 
* Main configuration file
* Plugin configuration file  

##### Main configuration file(config/config.yaml)
Main configuration file captures generic information relevant to compose executor. Details are outlined below.
```
launchtask:
   podmonitorinterval: 10000 # Periodic interval (in milisecond) at which pod is monitored. (Required)
   pullretry: 3              # Maximum retry count for pulling images. retry with backoff is 
                             # used on failure.(Optional, default value is 1.)
   maxretry: 3               # Maximum retry count for retrieving list of containers in a pod. 
                             # (Optional, defaults to 1) 
   retryinterval: 10000      # Interval between each cmd retry
                             # (Optional, defaults to 10s)
   timeout: 100000           # Timeout for pods get running. (Required)
plugins:
   pluginorder: general      # Define the order of plugins will be executed. If you register your 
                             # plugin with name "example", you will have "general,example" as pluginorder. 
                             # This is an important configuration to get pod running successfully. (Required)
foldername: poddata          # Folder to keep temporary files generated by plugins. 
                             # (Optional, default value is poddata)
cleanpod:                    # This section determines whether pod should be cleaned up or not if it becomes unhealthy
   unhealthy: true           # if set to true, clean up the pod when unhealthy.
   timeout: 20               # Timeout for stopping pod.
                             # (Optional, defaults to 10s)
   cleanvolumeandcontaineronmesoskill: true      # remove volumes and containers in the pod if pod is killed by mesos
                                                 # (Optional, defaults to false)
   cleanimageonmesoskill: true                   # remove images used by pod if pod is killed by mesos
                                                 # (Optional, defaults to false)
dockerdump:
   enable: true                                  # do docker dump if pod launch timeout.(Optional, default value is false)
   dumppath: /home/ubuntu                        # path to dump docker  
                                                 # (Optional, default value is /home/ubuntu)
   dockerpidfile: /var/run/docker.pid            # path of docker pid
                                                 # (Optional, default value is /var/run/docker.pid)
   containerpidfile: /run/docker/libcontainerd/docker-containerd.pid     # path of containerd pid
                                                 # (Optional, default value is /var/run/docker.pid)
   dockerlogpath: /var/log/upstart/docker.log    # path of docker log
                                                 # (Optional, default value is /var/log/upstart/docker.log)
dockercomposeverbose: true                       # enable verbose mode for each docker cmd
                                                 # (Optional, default value is false)
   
 
```
<!--
podmonitorinterval: Periodic interval (in milisecond) at which pod is monitored. (Required)

pullretry: Maximum retry count for pulling images. retry with backoff is used on failure.(Optional, default value is 1.)

maxretry: Maximum retry count for retrieving list of containers in a pod. (Optional, default value is 1)

timeout: Timeout for pods get running. (Required)

pluginorder: Define the order of plugins will be executed. If you register your plugin with name "example", you will have "general,example" as pluginorder. This will be a critical configuration to get pod running successfully. (Required)

healthcheck: health check is enabled or not in your plugins. (Required)

foldername: folder to keep temporary files generated by plugins. (Optional, default folder name is poddata)
-->

##### General Plugin configuration file(plugin/general/general.yaml)
Individual Plugin (such as General Plugin) configuration file caters to plugin relevant information. See details below:

```
infracontainer:
  image: $DOCKER_IMAGE_NAME
  container_name: $CONTAINER_NAME
  networks:
    pre_existing: true
    name : $NETWORK_NAME
    driver: $NETWORK_DRIVER
```
image: image for infrastructure container. The infrastructure container is used to collapse network namespace for pod so that all containers in pod share same ip. (Required)

container_name: container name for infrastructure container. (Required)

networks/pre_existing: defines whether network is pre-existing,  if so set to “true”. Otherwise, set to "false". (Required)

networks/name: Define the network name. (Optional, default value will be default)

networks/driver: Specify the network driver. (Optional, default value will be bridge)



