## Installing environment

This vagrant box helps you quickly create DCE environment which has mesos, marathon, aurora, golang1.7, docker, docker-compose and DCE installed. DCE will be compiled as a binary file called "executor" and you can find it at /home/vagrant. 

### Requirements

* Linux/Unix/Mac OS X
* VirtualBox
* Vagrant
* Git

### Steps
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

