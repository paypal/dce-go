package main

import (
	"fmt"
	"os"

	realis "github.com/paypal/gorealis"
	"github.com/paypal/gorealis/gen-go/apache/aurora"
)

func main() {
	r, err := realis.NewRealisClient(
		realis.SchedulerUrl("localhost"),
		realis.BasicAuth("aurora", "password"),
		realis.ThriftJSON(),
		realis.TimeoutMS(20000),
		realis.Debug(),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	job := realis.NewJob().
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
		AddURIs(true, false, "/dce-go/examples/sampleapp.tar.gz")

	fmt.Printf("Creating job %v\n", job.JobKey())

	resp, err := r.CreateJob(job)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(resp.String())

	monitor := &realis.Monitor{Client: r}
	if resp.ResponseCode == aurora.ResponseCode_OK {
		if ok, err := monitor.Instances(job.JobKey(), job.GetInstanceCount(), 5, 500); !ok || err != nil {
			_, err := r.KillJob(job.JobKey())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}
