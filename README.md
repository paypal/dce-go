# dce-go
[![godoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](http://godoc.org)

## Overview

dce-go project aims to enable Mesos frameworks to launch a pod of docker containers treating both Apache Mesos and docker as first class citizens. Kubernetes/K8 introduced the notion of a collection of docker containers that share namespaces and treat the collection as a single scaling unit. Brendan Burns talked about some design patterns/use cases for pods in [DockerCon'15](https://www.youtube.com/watch?v=Ph3t8jIt894).

Docker Compose is a cherished tool used in docker community that helps us model a collection of docker containers. The specification is very flexible. Furthermore, you can also model a pod collapsing namespaces (net, IPC , pid).

Composite containers representing an application is a common requirement for modular architecture. Composition requires co-location treating a set of containers as a single unit (aka pod) for scheduling. Sidecar, ambassador, adapter patterns use container pod. docker compose in docker community is an excellent way of defining a collection of containers and can be used to represent pod. Mesos on the other hand plays the critical role of a resource and cluster manager for large clusters. The native docker integration in Mesos can only launch a single container. In 1.1 Mesos has released Nested Container and Task Groups experimental feature to natively support a generic collection of tasks, not docker specific pods. However, frameworks need to change to support this and obviously the pod spec is separate than compose. universal containerizer with isolators is trying to model a different runtime than docker. docker swarm on the other hand as of 1.13/17.04 does not support local pods. compose-executor helps to immediately address the need of Mesos and docker users helping them launch a set of docker containers, aka pods as a single unit in Mesos without changing frameworks.

## Goal

The project goal is to model a pod of containers with docker-compose and launch it with your favorite Mesos frameworks like Marathon, Apache Aurora, etc. One does not need to switch to Kubernetes on Mesos if all that they are looking to do is launch pods (model pod like workloads). With network and storage plugins supported directly in Docker, one can model advanced pods supported through compose. Furthermore, instead of using some different spec to define pods, we wanted to build around the compose spec that is well accepted in the docker community. A developer can now write the pod spec once, run it locally in their laptop using compose and later seamlessly move into Mesos without having to modify the pod spec.

Running multiple pods on the same host may create many conflicts (containerId's , ports etc.). Executor takes care of resolving these conflicts.  A new docker-compose file resolving all the conflicts is generated. Each container is tagged with specific taskId and executorId and this is used to clean up containers via mesos hooks if executor is terminated. Cgroup is intruduced to limit, account for, and isolate the resource usage (CPU, memory, disk I/O, network, etc.) of a Pod.
 
dce-go is implemented in golang and provides a pluggable mechanism which gives developers more flexibilities to inject their custom logic. 
 


### Plugins
Pod is launched according to docker compose files provided by users. Docker compose files can be modified before pod is launched by dce-go. To allow developers implementing their own logic for customizing docker compose files based on specific requirements, pluggable structure is provided in dce-go. Please look into [How to develop](docs/how-to-develop.md) doc to understand how to implement plugins.

### Pod Modelling

##### cgroup hierarchy
dce-go mounts by default all the containers representing the pod under the parent mesos task cgroup. The memory subsystem use_hierarchy should be enabled for mesos cgroup. With this even if individual containers are not controlled, resources will be enforced as per the parent task limits. 

##### Infrastructure Container
Infrastructure container is the secret of how containers in a Pod can share the network namespace, including the IP address and network ports. We are not collapsing other namespaces like pid at this point in general plugin.

### Features
Implements mesos executor callbacks to maintain the lifecycle of a pod.
Massages compose file to add cgroup parent, mesos labels and edit certain sections to resolve any naming conflict etc
Collapses network namespace by default.
Provides pod monitor to not only kill entire pod on unexpected container exit but also when a container becomes unhealthy as per docker healthchecks.
Supports running multiple compose files.
Mesos Module provided to prevent pod leaks in rare case of executor crashes.
Provides plugins. 
Last but not the least any existing Mesos Frameworks like Aurora, Marathon etc can use DCE directly without making ANY framework changes.


### To start using dce-go
1. [Installing environment](docs/environment.md)
2. [How to use](docs/how-to-use.md)
    
### To start developing dce-go
1. [Installing environment](docs/environment.md)
2. [How to develop](docs/how-to-develop.md)

### Contributions
Contributions are always welcome. Please raise an issue so that the contribution may be discussed before it's made.

