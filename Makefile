#Licensed under the Apache License, Version 2.0 (the "License");         \
you may not use this file except in compliance with the License.         \
You may obtain a copy of the License at                                  \
http://www.apache.org/licenses/LICENSE-2.0                               \
Unless required by applicable law or agreed to in writing, software      \
distributed under the License is distributed on an "AS IS" BASIS,        \
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. \
See the License for the specific language governing permissions and      \
limitations under the License.

# set relative path to dockerfile
BUILD_PATH := /home/vagrant/go/src/github.com/paypal/dce-go/dce


all: build
	@echo "build binary file"

build:
	go build -o executor ${BUILD_PATH}/main.go
	sudo mv executor /home/vagrant

upload:
	sudo cp /home/vagrant/executor /usr/share/nginx/html