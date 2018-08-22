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
	"strings"
	"time"

	mesos "github.com/mesos/mesos-go/mesosproto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/paypal/dce-go/types"
)

// Define default values
const (
	CONFIG_File                          = "config"
	FOLDER_NAME                          = "foldername"
	HEALTH_CHECK                         = "healthcheck"
	POD_MONITOR_INTERVAL                 = "launchtask.podmonitorinterval"
	TIMEOUT                              = "launchtask.timeout"
	INFRA_CONTAINER                      = "infracontainer"
	PULL_RETRY                           = "launchtask.pullretry"
	MAX_RETRY                            = "launchtask.maxretry"
	RETRY_INTERVAL                       = "launchtask.retryinterval"
	NETWORKS                             = "networks"
	PRE_EXIST                            = "pre_existing"
	NETWORK_NAME                         = "name"
	NETWORK_DRIVER                       = "driver"
	CLEANPOD                             = "cleanpod"
	CLEAN_CONTAINER_VOLUME_ON_MESOS_KILL = "cleanpod.cleanvolumeandcontaineronmesoskill"
	CLEAN_IMAGE_ON_MESOS_KILL            = "cleanpod.cleanimageonmesoskill"
	CLEAN_FAIL_TASK                      = "cleanpod.cleanfailtask"
	DOCKER_COMPOSE_VERBOSE               = "dockercomposeverbose"
	SKIP_PULL_IMAGES                     = "launchtask.skippull"
	COMPOSE_TRACE                        = "launchtask.composetrace"
	DEBUG_MODE                           = "launchtask.debug"
	COMPOSE_HTTP_TIMEOUT                 = "launchtask.composehttptimeout"
	COMPOSE_STOP_TIMEOUT                 = "cleanpod.timeout"
	CONFIG_OVERRIDE_PREFIX               = "config."
)

// Read from default configuration file and set config as key/values
func init() {
	err := getConfigFromFile(CONFIG_File)

	if err != nil {
		log.Fatalf("Fail to retrieve data from file, err: %s\n", err.Error())
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

	setDefaultConfig(GetConfig())
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
	log.Printf("Set config : %s = %v", key, value)
	GetConfig().Set(key, value)
}

func GetConfigSection(section string) map[string]string {
	return viper.GetStringMapString(section)
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}

func setDefaultConfig(conf *viper.Viper) {
	conf.SetDefault(POD_MONITOR_INTERVAL, 10000)
	conf.SetDefault(COMPOSE_HTTP_TIMEOUT, 300)
	conf.SetDefault(MAX_RETRY, 3)
	conf.SetDefault(PULL_RETRY, 3)
	conf.SetDefault(RETRY_INTERVAL, 10000)
	conf.SetDefault(TIMEOUT, 500000)
	conf.SetDefault(COMPOSE_STOP_TIMEOUT, 10)
}

func GetAppFolder() string {
	folder := GetConfig().GetString(FOLDER_NAME)
	if folder == "" {
		return types.DEFAULT_FOLDER
	}
	return folder
}

func GetPullRetryCount() int {
	return GetConfig().GetInt(PULL_RETRY)
}

func GetLaunchTimeout() time.Duration {
	return time.Duration(GetConfig().GetInt(TIMEOUT))
}

func GetStopTimeout() int {
	return GetConfig().GetInt(COMPOSE_STOP_TIMEOUT)
}

func GetRetryInterval() time.Duration {
	return time.Duration(GetConfig().GetInt(RETRY_INTERVAL))
}

func GetMaxRetry() int {
	return GetConfig().GetInt(MAX_RETRY)
}

func GetNetwork() (types.Network, bool) {
	nmap, ok := GetConfig().GetStringMap(INFRA_CONTAINER)[NETWORKS].(map[string]interface{})
	if !ok {
		log.Println("networks section missing in config")
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

func EnableComposeTrace() bool {
	return GetConfig().GetBool(COMPOSE_TRACE)
}

func GetPollInterval() int {
	return GetConfig().GetInt(POD_MONITOR_INTERVAL)
}

func GetComposeHttpTimeout() int {
	return GetConfig().GetInt(COMPOSE_HTTP_TIMEOUT)
}

func EnableDebugMode() bool {
	return GetConfig().GetBool(DEBUG_MODE)
}

func IsService() bool {
	return GetConfig().GetBool(types.IS_SERVICE)
}

func CreateFileAppendMode(filename string) *os.File {

	File, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Errorf("Error in creating %v file", filename, err)
		return os.Stdout
	}
	return File
}

// OverrideConfig gets labels with override prefix "config." and override configs with value of label
// Checking labels contain key word "config." instead of prefix since different framework will add different prefix for labels
func OverrideConfig(taskInfo *mesos.TaskInfo) {
	labelsList := taskInfo.GetLabels().GetLabels()

	for _, label := range labelsList {
		if strings.Contains(label.GetKey(), CONFIG_OVERRIDE_PREFIX) {
			parts := strings.SplitAfterN(label.GetKey(), CONFIG_OVERRIDE_PREFIX, 2)
			if len(parts) == 2 && GetConfig().IsSet(parts[1]) {
				GetConfig().Set(parts[1], label.GetValue())
				log.Infof("override config %s with %s", parts[1], label.GetValue())
			}
		}
	}
}
