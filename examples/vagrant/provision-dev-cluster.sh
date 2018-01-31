#!/bin/bash -ex
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
echo deb https://apt.dockerproject.org/repo ubuntu-trusty main > /etc/apt/sources.list.d/docker.list
apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv E56151BF
echo deb http://repos.mesosphere.com/ubuntu trusty main > /etc/apt/sources.list.d/mesosphere.list

apt-get update -q --fix-missing
apt-get -qy install software-properties-common
add-apt-repository ppa:george-edison55/cmake-3.x
apt-get update -q
apt-cache policy cmake

add-apt-repository ppa:openjdk-r/ppa
apt-get update
apt-get -y install                         \
   wget                                    \
   tar                                     \
   bison                                   \
   curl                                    \
   git                                     \
   openjdk-8-jdk                           \
   docker-engine                           \
   python-dev                              \
   zookeeperd                              \
   python-pip                              \
   maven                                   \
   build-essential                         \
   autoconf                                \
   automake                                \
   ca-certificates                         \
   protobuf-compiler                       \
   libprotobuf-dev                         \
   make                                    \
   python                                  \
   python2.7                               \
   libpython-dev                           \
   python-dev                              \
   python-protobuf                         \
   python-setuptools                       \
   unzip                                   \
   --no-install-recommends

sudo echo 1 > /sys/fs/cgroup/memory/memory.use_hierarchy

update-alternatives --set java /usr/lib/jvm/java-8-openjdk-amd64/jre/bin/java

readonly IP_ADDRESS=192.168.33.8
readonly MESOS_VERSION=1.0.0-2.0.89
readonly MESOS_MODULE=v1.0.0
readonly MARATHON_VERSION=1.1.2-1.0.482

function install_go {

  wget -c "https://storage.googleapis.com/golang/go1.9.2.linux-amd64.tar.gz" "--no-check-certificate"
  tar -xvf go1.9.2.linux-amd64.tar.gz
  sudo mv go /usr/local

  export GOROOT=/usr/local/go
  export GOPATH=/home/vagrant/go
  export PATH=$PATH:$GOPATH/bin:$GOROOT/bin

  sed -e "\$aexport PATH=$PATH" /home/vagrant/.bashrc>/home/vagrant/.bashrc.new
  mv /home/vagrant/.bashrc.new /home/vagrant/.bashrc
  chmod 777 /home/vagrant/.bashrc

  sudo mkdir -p "$GOPATH/bin" "$GOPATH/src/github.com/paypal"

  curl https://glide.sh/get | sh

}

function install_mesos {
  apt-get -y install mesos=${MESOS_VERSION}.ubuntu1404
}

function prepare_extra {
  mkdir -p /etc/mesos-slave
  mkdir -p /etc/mesos-master
  mkdir -p /output/usr/local/lib/mesos
  mkdir -p /etc/docker || true
  cp /vagrant/examples/vagrant/config/daemon.json /etc/docker/
  cp /vagrant/examples/vagrant/mesos_config/etc_mesos-slave/* /etc/mesos-slave
  cp /vagrant/examples/vagrant/mesos_config/etc_mesos-master/* /etc/mesos-master
  cp /vagrant/examples/vagrant/mesos-module/${MESOS_MODULE}/* /output/usr/local/lib/mesos/
  cp /vagrant/examples/vagrant/marathon/marathon /etc/default/marathon
}

function install_aurora {
  wget -c https://apache.bintray.com/aurora/ubuntu-trusty/aurora-scheduler_0.16.0_amd64.deb
  sudo dpkg -i aurora-scheduler_0.16.0_amd64.deb
  sudo stop aurora-scheduler
  sudo -u aurora mkdir -p /var/lib/aurora/scheduler/db
  sudo -u aurora mesos-log initialize --path=/var/lib/aurora/scheduler/db
  sudo cp /vagrant/examples/vagrant/config/aurora-scheduler.conf /etc/init
  sudo start aurora-scheduler
}

function install_marathon {
  apt-get -y install marathon=${MARATHON_VERSION}.ubuntu1404
}

function install_docker_compose {
  pip install docker-compose
}

function build_docker_compose_executor {
  sudo ln -sf /vagrant $GOPATH/src/github.com/paypal/dce-go
  sudo chmod 777 $GOPATH/src/github.com/paypal/dce-go
  cd $GOPATH/src/github.com/paypal/dce-go
  glide install
  go build -o executor $GOPATH/src/github.com/paypal/dce-go/dce/main.go
  mv executor /home/vagrant
  sed -i.bkp '$a export GOROOT=/usr/local/go;export GOPATH=/home/vagrant/go;export PATH=$PATH:$GOPATH/bin:$GOROOT/bin' /home/vagrant/.bashrc
  . /home/vagrant/.bashrc
}

function install_cluster_config {
  mkdir -p /etc/marathon
  ln -sf /vagrant/examples/vagrant/clusters.json /etc/marathon/clusters.json
}

function install_ssh_config {
  cat >> /etc/ssh/ssh_config <<EOF
# Allow local ssh w/out strict host checking
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
EOF
}

function configure_netrc {
  cat > /home/vagrant/.netrc <<EOF
machine $(hostname -f)
login aurora
password secret
EOF
  chown vagrant:vagrant /home/vagrant/.netrc
}

function sudoless_docker_setup {
  gpasswd -a vagrant docker
  service docker restart
}

function start_services {
  #Executing true on failure to please bash -e in case services are already running
  start zookeeper    || true
  start mesos-master || true
  start mesos-slave  || true
  start marathon || true
  start aurora-scheduler || true
}

function configure_nginx {

  sudo apt-get -y install nginx
  sudo service nginx start

  # set up nginx server
  sudo cp /vagrant/examples/vagrant/nginx/nginx.conf /etc/nginx/sites-available/site.conf
  sudo chmod 644 /etc/nginx/sites-available/site.conf
  sudo ln -s /etc/nginx/sites-available/site.conf /etc/nginx/sites-enabled/site.conf
  sudo service nginx restart

  # clean /var/www
  sudo rm -Rf /var/www

  # symlink /var/www => /vagrant
  #ln -s /vagrant /var/www

  sudo cp /vagrant/examples/vagrant/config/config.yaml /usr/share/nginx/html
  sudo mkdir -p /usr/share/nginx/html/example
  sudo cp /vagrant/examples/vagrant/config/config_example.yaml /usr/share/nginx/html/example/config.yaml
  sudo cp /vagrant/examples/vagrant/config/general.yaml /usr/share/nginx/html
  sudo cp /home/vagrant/executor /usr/share/nginx/html
  sudo cp -r /vagrant/examples/sampleapp /usr/share/nginx/html/
  cd /usr/share/nginx/html
  sudo tar -czvf app.tar.gz sampleapp
}

function install_mesos_modules {
  cd /vagrant
  sudo docker build -t mesos-build -f Dockerfile-mesos-modules --build-arg MESOS_VERSION=1.0.0 .
  sudo docker run --rm -v /output:/output mesos-build make install DESTDIR=/output
}


prepare_extra
install_go
#install_mesos_modules
install_mesos
install_aurora
install_marathon
install_docker_compose
install_cluster_config
install_ssh_config
start_services
configure_netrc
sudoless_docker_setup
build_docker_compose_executor
configure_nginx
