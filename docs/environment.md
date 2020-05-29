## Installing environment

This vagrant box helps you quickly create DCE environment which has mesos, marathon, aurora, golang1.7, docker, docker-compose and DCE installed. DCE will be compiled as a binary file called "executor" and you can find it at /home/vagrant. 

### Requirements

* Linux/Unix/Mac OS X
* VirtualBox
* Vagrant
* Git

### Vagrant Setup
1. [Install VirtualBox](https://www.virtualbox.org/wiki/Downloads)

2. [Install Vagrant](https://www.vagrantup.com/downloads.html)
3. Git clone this repository
  ```
  git clone https://github.com/paypal/dce-go
  
  cd dce-go
  ```
4. Start vagrant box

  ```
  vagrant up
  ```
5. Validate installation

  ```
  mesos endpoint : http://192.168.33.8:5050
  
  marathon endpoint: http://192.168.33.8:8080/ui
  
  aurora endpoint: http://192.168.33.8:8081
  
  ```
6. ssh to the vagrant box

  ```
  vagrant ssh
  ```
7. Next: [How to use](how-to-use.md)

### Docker-Compose Setup

1. [Install docker](https://docs.docker.com/get-docker/)

1. [Install docker-compose](https://docs.docker.com/compose/install/)

1. Clone the git repository: `$ git clone https://github.com/paypal/dce-go`

1. Build the executor: `$ GOOS=linux GOARCH=amd64 go build -o dce-go-linux-amd64 dce/main.go`

1. Use docker-compose to bring up environment: `$ docker-compose up -d`

The base set up is now ready. Aurora's and Mesos' Web UI should now be available.

They can be found at:

* Aurora: http://localhost:8081 (Mac) or http://192.168.33.7:8081 (Linux)
* Mesos: http://localhost:5050 (Mac) or http://192.168.33.3:5050 (Linux)


#### Running a sample job in docker-compose environment

1. Create a tar.gz from our sample application:

`$  tar -czf examples/sampleapp.tar.gz -C examples/ sampleapp`

2. Use gorealis to create a job in Aurora:

`$ go run examples/docker-compose/create.go`

3. Navigate to the Aurora or Mesos UI to see progress of job creation.

4. If needed, the job may now be killed using gorealis:

`$ go run examples/docker-compose/kill.go`

