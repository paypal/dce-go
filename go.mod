module github.com/paypal/dce-go

go 1.13

require (
	git.apache.org/thrift.git v0.0.0-20161221203622-b2a4d4ae21c7
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/kr/text v0.2.0 // indirect
	github.com/mesos/mesos-go v0.0.3-0.20161004192122-7228b13084ce
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/paypal/gorealis v1.1.0
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20201109172640-a11eb1b685be // indirect
	golang.org/x/sys v0.0.0-20201214210602-f9fddec55a1e // indirect
	golang.org/x/text v0.3.5 // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/ini.v1 v1.57.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect

)

replace (
	git.apache.org/thrift.git v0.0.0-20161221203622-b2a4d4ae21c7 => github.com/rdelval/thrift v0.0.0-20161221203622-b2a4d4ae21c7
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig => github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig v0.14.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk => github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk v0.14.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver => github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver v0.14.0
)