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
        Name("sampleapp")

    fmt.Printf("Killing job %v\n", job.JobKey())
    resp, err := r.KillJob(job.JobKey())
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    monitor := &realis.Monitor{Client: r}
    if resp.ResponseCode == aurora.ResponseCode_OK {
        if ok, err := monitor.Instances(job.JobKey(), 0, 5, 50); !ok || err != nil {
            fmt.Println("Unable to kill all instances of job")
            os.Exit(1)
        }
    }
    fmt.Println(resp.String())
}
