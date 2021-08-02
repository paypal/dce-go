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

//Monitor pod status
package monitor

import (
	"context"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/plugin"
	"github.com/paypal/dce-go/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// MonitorPoller inspects pod status periodically
func MonitorPoller(ctx context.Context) (types.PodStatus, error) {
	logger := log.WithFields(log.Fields{
		"func": "monitor.MonitorPoller",
	})

	logger.Println("====================Pod Monitor Poller====================")

	name := config.GetConfig().GetString("podMonitor.monitorName")
	monitor := plugin.Monitors.Lookup(name)
	if monitor == nil {
		return types.POD_FAILED, errors.Errorf("monitor plugin %s doesn't exist", name)
	}
	return monitor.Start(ctx)
}
