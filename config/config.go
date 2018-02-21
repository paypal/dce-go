/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package config helps plugins extensions to easily set config as key/value by reading from configuration file
// Package config also provides setter and getter functionality

package config

import (
	"io/ioutil"

	"os"
	"path/filepath"

	"strconv"

	"fmt"

	"time"

	"github.com/paypal/dce-go/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Define default values
const (
	CONFIG_File                          = "config"
	FOLDER_NAME                          = "foldername"
	LAUNCH_TASK                          = "launchtask"
	HEALTH_CHECK                         = "healthcheck"
	POD_MONITOR_INTERVAL                 = "podmonitorinterval"
	TIMEOUT                              = "timeout"
	INFRA_CONTAINER                      = "infracontainer"
	PULL_RETRY                           = "pullretry"
	MAX_RETRY                            = "maxretry"
	RETRY_INTERVAL                       = "retryinterval"
	NETWORKS                             = "networks"
	PRE_EXIST                            = "pre_existing"
	NETWORK_NAME                         = "name"
	NETWORK_DRIVER                       = "driver"
	CLEANPOD                             = "cleanpod"
	CLEAN_CONTAINER_VOLUME_ON_MESOS_KILL = "cleanvolumeandcontaineronmesoskill"
	CLEAN_IMAGE_ON_MESOS_KILL            = "cleanimageonmesoskill"
	DOCKER_COMPOSE_VERBOSE               = "dockercomposeverbose"
	SKIP_PULL_IMAGES                     = "launchtask.skippull"
)

// Read from default configuration file and set config as key/values
func init() {
	err := getConfigFromFile(CONFIG_File)

	if err != nil {
		log.Fatalf("Fail to retreive data from file, err: %s\n", err.Error())
	}
}

// Plugin extensions could merge configuration using ConfigInit with configuration file
func ConfigInit(pluginConfig string) {
	log.Printf("Plugin Merge Config : %s", pluginConfig)

	file, err := os.Open(pluginConfig)
	if err != nil {
		log.Fatalf("Fail to open file, err: %s\n", err.Error())
	}

	err = viper.MergeConfig(file)
	if err != nil {
		log.Fatalf("Fail to merge config, err: %s\n", err.Error())
	}
}

func getConfigFromFile(cfgFile string) error {
	// Set config name
	viper.SetConfigName(cfgFile)

	// Set config type
	viper.SetConfigType("yaml")

	// Add path for searching config files including plugins'
	viper.AddConfigPath(".")
	abs_path, _ := filepath.Abs(".")
	viper.AddConfigPath(filepath.Join(filepath.Dir(filepath.Dir(abs_path)), "config"))
	dirs, _ := ioutil.ReadDir("./")
	for _, f := range dirs {
		if f.IsDir() {
			viper.AddConfigPath(f.Name())
		}
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("No config file loaded, err: %s\n", err.Error())
		return err
	}
	return nil
}

func SetConfig(key string, value interface{}) {
	log.Println(fmt.Sprintf("Set config : %s = %v", key, value))
	GetConfig().Set(key, value)
}

func GetConfigSection(section string) map[string]string {
	return viper.GetStringMapString(section)
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}

func GetAppFolder() string {
	folder := GetConfig().GetString(FOLDER_NAME)
	if folder == "" {
		return types.DEFAULT_FOLDER
	}
	return folder
}

func GetPullRetryCount() int {
	retry := GetConfigSection(LAUNCH_TASK)[PULL_RETRY]
	if retry == "" {
		return 1
	}

	_retry, err := strconv.Atoi(retry)
	if err != nil {
		log.Println("Error converting retry count from string to int : ", err.Error())
		return 1
	}

	if _retry <= 0 {
		return 1
	}
	return _retry
}

func GetLaunchTimeout() time.Duration {
	timeout := GetConfigSection(LAUNCH_TASK)[TIMEOUT]
	if timeout == "" {
		log.Warningln("pod timeout doesn't set in config...timeout will be set as 500s")
		return time.Duration(500000)
	}
	t, err := strconv.Atoi(timeout)
	if err != nil {
		log.Errorf("Error converting timeout from string to int : %s...timeout will be set as 500s\n", err.Error())
		return time.Duration(500000)
	}
	return time.Duration(t)
}

func GetStopTimeout() string {
	timeout := GetConfigSection(CLEANPOD)[TIMEOUT]
	if timeout == "" {
		log.Warningln("pod timeout doesn't set in config...timeout will be set as 10s")
		return "10"
	}
	return timeout
}

func GetRetryInterval() time.Duration {
	interval := GetConfigSection(LAUNCH_TASK)[RETRY_INTERVAL]
	if interval == "" {
		log.Warningln("retry interval doesn't set in config...timeout will be set as 10s")
		return time.Duration(10000)
	}
	t, err := strconv.Atoi(interval)
	if err != nil {
		log.Fatalf("Error converting timeout from string to int : %s\n", err.Error())
	}
	return time.Duration(t)
}

func GetMaxRetry() int {
	retry := GetConfigSection(LAUNCH_TASK)[MAX_RETRY]
	if retry == "" {
		log.Warningln("maxretry setting missing in config...setting to 1")
		return 1
	}
	i, err := strconv.Atoi(retry)
	if err != nil {
		log.Errorf("Error converting retry from string to int : %s...setting max retry to 1\n", err.Error())
		return 1
	}
	return i
}

func GetNetwork() (types.Network, bool) {
	nmap, ok := GetConfig().GetStringMap(INFRA_CONTAINER)[NETWORKS].(map[string]interface{})
	if !ok {
		log.Println("networks section missing in configration file")
		return types.Network{}, false
	}

	if nmap[PRE_EXIST] == nil {
		nmap[PRE_EXIST] = false
	}

	if nmap[NETWORK_NAME] == nil {
		nmap[NETWORK_NAME] = ""
	}

	if nmap[NETWORK_DRIVER] == nil {
		nmap[NETWORK_DRIVER] = ""
	}

	network := types.Network{
		PreExist: nmap[PRE_EXIST].(bool),
		Name:     nmap[NETWORK_NAME].(string),
		Driver:   nmap[NETWORK_DRIVER].(string),
	}
	return network, true
}

func EnableVerbose() bool {
	return GetConfig().GetBool(DOCKER_COMPOSE_VERBOSE)
}

func SkipPullImages() bool {
	return GetConfig().GetBool(SKIP_PULL_IMAGES)
}
