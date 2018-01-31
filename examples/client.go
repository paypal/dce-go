/**
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	_ "io/ioutil"
	"os"

	"github.com/paypal/gorealis"
	"github.com/paypal/gorealis/gen-go/apache/aurora"
	_"github.com/paypal/gorealis/response"
)

func main() {
	cmd := flag.String("cmd", "", "Job request type to send to Aurora Scheduler")
	executor := flag.String("executor", "thermos", "Executor to use")
	url := flag.String("url", "", "URL at which the Aurora Scheduler exists as [url]:[port]")
	username := flag.String("username", "aurora", "Username to use for authorization")
	password := flag.String("password", "secret", "Password to use for authorization")
	flag.Parse()

	var job realis.Job
	var err error
	var monitor *realis.Monitor
	var r realis.Realis

	r, err = realis.NewRealisClient(realis.SchedulerUrl(*url), realis.BasicAuth(*username, *password), realis.ThriftJSON(), realis.TimeoutMS(20000))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	monitor = &realis.Monitor{r}

	switch *executor {
	case "compose":
		job = realis.NewJob().
			Environment("prod").
			Role("vagrant").
			Name("sampleapp").
			ExecutorName("docker-compose-executor").
			ExecutorData("{}").
			CPU(0.25).
			RAM(256).
			Disk(100).
			IsService(true).
			InstanceCount(1).
			AddPorts(4).
			AddLabel("fileName", "sampleapp/docker-compose.yml,sampleapp/docker-compose-healthcheck.yml").
			AddURIs(true, false, "http://192.168.33.8/app.tar.gz")
		break
	case "none":
		job = realis.NewJob().
			Environment("prod").
			Role("vagrant").
			Name("docker_as_task").
			CPU(1).
			RAM(64).
			Disk(100).
			IsService(true).
			InstanceCount(1).
			AddPorts(1)
		break
	default:
		fmt.Println("Only thermos, compose, and none are supported for now")
		os.Exit(1)
	}

	switch *cmd {
	case "create":
		fmt.Println("Creating job")
		resp, err := r.CreateJob(job)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(resp.String())

		if resp.ResponseCode == aurora.ResponseCode_OK {
			if ok, err := monitor.Instances(job.JobKey(), job.GetInstanceCount(), 5, 500); !ok || err != nil {
				_, err := r.KillJob(job.JobKey())
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
		break
	case "createDocker":
		fmt.Println("Creating a docker based job")
		container := realis.NewDockerContainer().Image("python:2.7").AddParameter("network", "host")
		job.Container(container)
		resp, err := r.CreateJob(job)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(resp.String())

		if resp.ResponseCode == aurora.ResponseCode_OK {
			if ok, err := monitor.Instances(job.JobKey(), job.GetInstanceCount(), 10, 300); !ok || err != nil {
				_, err := r.KillJob(job.JobKey())
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		}
		break
	case "kill":
		fmt.Println("Killing job")
		resp, err := r.KillJob(job.JobKey())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if resp.ResponseCode == aurora.ResponseCode_OK {
			if ok, err := monitor.Instances(job.JobKey(), 0, 5, 50); !ok || err != nil {
				fmt.Println("Unable to kill all instances of job")
				os.Exit(1)
			}
		}
		fmt.Println(resp.String())
		break
	case "restart":
		fmt.Println("Restarting job")
		resp, err := r.RestartJob(job.JobKey())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(resp.String())
		break
	case "liveCount":
		fmt.Println("Getting instance count")

		live, err := r.GetInstanceIds(job.JobKey(), aurora.LIVE_STATES)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Number of live instances: ", len(live))
		break
	case "activeCount":
		fmt.Println("Getting instance count")

		live, err := r.GetInstanceIds(job.JobKey(), aurora.ACTIVE_STATES)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Number of live instances: ", len(live))
		break
	case "flexUp":
		fmt.Println("Flexing up job")

		numOfInstances := int32(2)
		resp, err := r.AddInstances(aurora.InstanceKey{job.JobKey(), 0}, numOfInstances)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if resp.ResponseCode == aurora.ResponseCode_OK {
			if ok, err := monitor.Instances(job.JobKey(), job.GetInstanceCount()+numOfInstances, 5, 500); !ok || err != nil {
				fmt.Println("Flexing up failed")
			}
		}
		fmt.Println(resp.String())
		break
	case "taskConfig":
		fmt.Println("Getting job info")
		config, err := r.FetchTaskConfig(aurora.InstanceKey{job.JobKey(), 0})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		print(config.String())
		break
	default:
		fmt.Println("Command not supported")
		os.Exit(1)
	}
}