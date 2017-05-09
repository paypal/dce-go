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

package plugin

func GetOrderedExtpoints(plugins []string) []ComposePlugin {
	return ComposePlugins.Select(plugins)
}

func GetReverseOrderedExtpoints(plugins []ComposePlugin) []ComposePlugin {
	for i := 0; i < len(plugins)/2; i++ {
		j := len(plugins) - i - 1
		plugins[i], plugins[j] = plugins[j], plugins[i]
	}
	return plugins
}
