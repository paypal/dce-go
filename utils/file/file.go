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

package file

import (
	"bufio"
	"bytes"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	mesos "github.com/mesos/mesos-go/mesosproto"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/paypal/dce-go/config"
	"github.com/paypal/dce-go/types"
	"github.com/paypal/dce-go/utils/pod"
)

const (
	FILE_DELIMITER = ","
	YAML_SEPARATOR = "#---#"
	FILE_POSTFIX   = "-generated.yml"
	PATH_DELIMITER = "/"
	MAP_DELIMITER  = "="
	TraceFolder    = "composetrace"
)

type EditorFunc func(serviceName string, taskInfo *mesos.TaskInfo, executorId string, taskId string, containerDetails map[interface{}]interface{}, ports *list.Element) (map[interface{}]interface{}, *list.Element, error)

// Get required file list from label of fileName in taskInfo
func GetFiles(taskInfo *mesos.TaskInfo) ([]string, error) {
	log.Println("====================Retrieve  compose file list from fileName label====================")

	filelist := pod.GetLabel("fileName", taskInfo)
	if filelist == "" {
		err := errors.New("missing label fileName")
		log.Errorln(err)
		return nil, err
	}

	var files []string
	for _, file := range strings.Split(filelist, FILE_DELIMITER) {
		files = append(files, strings.TrimSpace(file))
	}

	log.Println("Required file list : ", files)
	return files, nil
}

// Get plugin order from label of pluginorder in taskInfo
func GetPluginOrder(taskInfo *mesos.TaskInfo) ([]string, error) {
	log.Println("====================Get plugin order====================")

	pluginList := pod.GetLabel(types.PLUGIN_ORDER, taskInfo)
	if pluginList == "" {
		err := errors.New("Missing label pluginorder")
		return nil, err
	}

	var plugins []string
	for _, plugin := range strings.Split(pluginList, FILE_DELIMITER) {
		plugins = append(plugins, plugin)
	}

	log.Println("Plugin Order : ", plugins)
	return plugins, nil
}

// Get downloaded file paths from uris
func GetYAML(taskInfo *mesos.TaskInfo) []string {
	log.Println("====================Get compose file from URI====================")
	var files []string
	uris := taskInfo.Executor.Command.GetUris()
	for _, uri := range uris {
		arr := strings.Split(uri.GetValue(), "/")
		name := arr[len(arr)-1]
		GetDirFilesRecv(name, &files)
	}
	log.Println("Compose file from URI: ", files)
	return files
}

// Get path compose file
// Since user may upload a tar ball which including all the compose files.
// In case they have different depth of folders to keep compose files, GetDirFilesRecv help to get the complete path of
// compose file
func GetDirFilesRecv(dir string, files *[]string) {
	if d, _ := os.Stat(dir); !d.IsDir() && (strings.HasSuffix(dir, ".yml") || strings.HasSuffix(dir, ".yaml") || dir == "yaml") {
		*files = append(*files, dir)
		return
	}
	dirs, err := ioutil.ReadDir(dir)
	if err != nil || dirs == nil || len(dirs) == 0 {
		log.Printf("%s is not a directory : %v", dir, err)
		return
	}
	for _, f := range dirs {
		if !f.IsDir() {
			if strings.Contains(f.Name(), ".yml") {
				*files = append(*files, dir+"/"+f.Name())
			}
		} else {
			GetDirFilesRecv(dir+"/"+f.Name(), files)
		}
	}
}

// search a file
// return path of a file
func SearchFile(root, file string) string {
	var filePath string
	filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.Name() == file {
			filePath = path
			return nil
		}
		return nil
	})
	return filePath
}

// check if "first" is subset of "second"
func IsSubset(first, second []string) bool {
	log.Println("subset : ", first)
	log.Println("set : ", second)
	set := make(map[string]int)
	for _, value := range second {
		set[value] += 1
	}

	for _, value := range first {
		if count, found := set[value]; !found {
			return false
		} else if count < 1 {
			return false
		} else {
			set[value] = count - 1
		}
	}

	return true
}

// check if file exist
func CheckFileExist(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Printf("File %s does not exist", file)
		return false
	}
	return true
}

// Add taskId as prefix
func PrefixTaskId(taskId string, session string) string {
	if strings.HasPrefix(session, taskId) {
		return session
	}
	return taskId + "_" + session
}

// Generate file with provided name and write data into it.
func WriteToFile(file string, data []byte) (string, error) {
	if !strings.Contains(file, config.GetAppFolder()) {
		file = FolderPath(strings.Fields(file))[0]
	}

	log.Printf("Write to file : %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Errorf("Error creating file %v", err)
		return "", err
	}

	_, err = f.Write(data)
	if err != nil {
		log.Errorln("Error writing into file : ", err.Error())
		return "", err
	}
	return f.Name(), nil
}

func DeleteFile(file string) error {
	if !strings.Contains(file, config.GetAppFolder()) {
		file = FolderPath(strings.Fields(file))[0]
	}
	return os.Remove(file)
}

func WriteChangeToFiles(ctx context.Context) error {
	filesMap := ctx.Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	for file := range filesMap {
		content, _ := yaml.Marshal(filesMap[file])
		_, err := WriteToFile(file.(string), content)
		if err != nil {
			return err
		}
	}
	return nil
}

func DumpPluginModifiedComposeFiles(ctx context.Context, plugin string, pluginOrder int) {
	filesMap := ctx.Value(types.SERVICE_DETAIL).(types.ServiceDetail)
	for file := range filesMap {
		content, _ := yaml.Marshal(filesMap[file])
		fParts := strings.Split(file.(string), PATH_DELIMITER)
		if len(fParts) < 2 {
			log.Printf("Skip dumping modified compose file by plugin %s, since file name is invalid %s", plugin, file)
			return
		}
		_, err := WriteToFile(fmt.Sprintf("%s/%s/%s-%s-%d.yml", fParts[0], TraceFolder, fParts[1], plugin, pluginOrder), content)
		if err != nil {
			log.Printf("Failed dumping modified compose file by plugin %s", plugin)
		}
	}
}

func OverwriteFile(file string, data []byte) {
	log.Printf("Over-write file: %s\n", file)

	os.Remove(file)

	f, err := os.Create(file)
	if err != nil {
		log.Errorln("Error creating file")
	}

	_, err = f.Write(data)
	if err != nil {
		log.Errorln("Error writing into file : ", err.Error())
	}
}

//Split a large file into a number of smaller files by file separator
func SplitYAML(file string) ([]string, error) {
	logger := log.WithFields(log.Fields{
		"File": file,
		"Func": "SplitYAML",
	})

	logger.Println("Start split downloaded compose file")
	var names []string
	if _, err := os.Stat(file); os.IsNotExist(err) {
		logger.Printf("%s doesn't exit\n", file)
		return nil, err
	}

	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(dat)))
	scanner.Split(SplitFunc)
	for scanner.Scan() {
		splitData := scanner.Text()
		logger.Printf("Split data : %s\n", splitData)

		// Get split compose file name from split data
		// If name isn't found, skip writing split data into a separate file
		name := strings.TrimLeft(getYAMLDocumentName(splitData, "#.*yml"), "#")
		if name == "" {
			logger.Println("No file name found from split data")
			continue
		}
		fileName, err := WriteToFile(name, []byte(splitData))
		if err != nil {
			logger.Errorf("Error writing files %s : %v\n", fileName, err)
			return nil, err
		}
		names = append(names, fileName)
	}

	// If file doesn't need to be split, just return the file
	if len(names) == 0 {
		names = append(names, file)
	}

	logger.Printf("After split file, get file names: %s\n", names)
	return FolderPath(names), nil
}

// splitYAMLDocument is a bufio.SplitFunc for splitting YAML streams into individual documents.
// This SplitFunc code is from K8s utils, since the yaml serperator is different, it can't be reused directly.
func SplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(YAML_SEPARATOR))
	if i := bytes.Index(data, []byte(YAML_SEPARATOR)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// get yaml file name from a file using a defined pattern
func getYAMLDocumentName(data, pattern string) string {
	if len(data) <= 0 {
		return ""
	}
	r, _ := regexp.Compile(pattern)
	name := r.FindString(data)
	return name
}

//ReplaceElement does replace element in array/map
func ReplaceElement(i interface{}, old string, new string) interface{} {
	if array, ok := i.([]interface{}); ok {
		index, err := IndexArrayRegex(array, old)
		if err != nil || index == -1 {
			return array
		}
		array[index] = new
		return array

	}

	if m, ok := i.(map[interface{}]interface{}); ok {
		_, exit := m[old]
		if exit {
			m[old] = new
		}
		return m

	}

	log.Println("Only support replacing elements in array and map")

	return i
}

//AppendElement does append element in array/map
//Element will be overwrite if it already exist
//Using regular expression to match the element
func AppendElement(i interface{}, old string, new string) interface{} {
	if array, ok := i.([]interface{}); ok {
		index, err := IndexArrayRegex(array, old)
		if err != nil || index == -1 {
			array = append(array, new)
			return array
		} else {
			array[index] = new
		}
		return array

	}

	if m, ok := i.(map[interface{}]interface{}); ok {
		m[old] = new
		return m
	}

	log.Println("Only support appending elements in array and map")

	return i
}

func IndexArrayRegex(array []interface{}, expr string) (int, error) {
	r, err := regexp.Compile(expr)
	if err != nil {
		log.Errorf("Error compile expression: %v\n", err)
		return -1, err
	}
	for i, a := range array {
		if r.MatchString(a.(string)) {
			return i, nil
		}
	}
	return -1, err
}

// get index of an element in array
func IndexArray(array []interface{}, element string) (int, error) {
	for i, e := range array {
		if e == element {
			return i, nil
		}
	}
	return -1, errors.New("Element missing in list")
}

func SearchInArray(array []interface{}, key string) string {
	for _, e := range array {
		if s := strings.Split(e.(string), MAP_DELIMITER); len(s) > 1 && s[0] == key {
			return s[1]
		}
	}
	return ""
}

// []string to []interface{}
func FormatInterfaceArray(s []string) []interface{} {
	t := make([]interface{}, len(s))
	for i, v := range s {
		t[i] = v
	}
	return t
}

// generate directories
func GenerateFileDirs(paths []string) error {
	log.Println("Generate Folders (0777): ", paths)
	for _, path := range paths {

		err := os.MkdirAll(path, 0777)
		if err != nil {
			log.Println("Error creating directory : ", err.Error())
			return err
		}

		os.Chmod(path, 0777)
	}
	return nil
}

func FolderPath(filenames []string) []string {
	if config.GetConfig().GetBool(types.NO_FOLDER) {
		return filenames
	}

	folder := config.GetAppFolder()

	for i, filename := range filenames {
		if !strings.Contains(filename, config.GetAppFolder()) {
			filenames[i] = strings.TrimSpace(folder) + PATH_DELIMITER + filename
		}
	}

	return filenames
}

func DeFolderPath(filepaths []string) []string {
	filenames := make([]string, len(filepaths))
	for _, file := range filepaths {
		filenames = append(filenames, filepath.Base(file))
	}
	return filenames
}

func ParseYamls(files *[]string) (map[interface{}](map[interface{}]interface{}), error) {
	res := make(map[interface{}](map[interface{}]interface{}))
	for _, file := range *files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Error reading file %s : %v", file, err)
		}
		m := make(map[interface{}]interface{})
		err = yaml.Unmarshal(data, &m)
		if err != nil {
			log.Errorf("Error unmarshalling %v", err)
		}
		res[FolderPath(strings.Fields(file))[0]] = m
	}
	return res, nil
}

// GenerateAppFolder does generate app folder and copy compose files exist in fileName label
// into folder
func GenerateAppFolder() error {
	folder := config.GetAppFolder()
	if folder == "" {
		folder = types.DEFAULT_FOLDER
		config.SetConfig(config.FOLDER_NAME, types.DEFAULT_FOLDER)
	}

	// Append run id to folder name
	path, _ := filepath.Abs("")
	dirs := strings.Split(path, PATH_DELIMITER)
	folder = strings.TrimSpace(fmt.Sprintf("%s_%s", folder, dirs[len(dirs)-1]))

	// Folder to keep all compose files generated by plugins
	traceFolder := fmt.Sprintf("%s/%s", folder, TraceFolder)

	// Generate directory
	folders := []string{strings.TrimSpace(folder), traceFolder}
	err := GenerateFileDirs(folders)
	if err != nil {
		log.Println("Error generating file dirs: ", err.Error())
		return err
	}

	config.GetConfig().Set(config.FOLDER_NAME, folder)

	// Copy compose files into pod folder
	for i, file := range pod.ComposeFiles {
		path := strings.Split(file, PATH_DELIMITER)
		dest := FolderPath(strings.Fields(path[len(path)-1]))[0]
		err = CopyFile(file, dest)
		if err != nil {
			log.Printf("Copy file %s into pod folder %v", file, err)
		}
		pod.ComposeFiles[i] = dest
	}

	log.Printf("compose file list: %s\n", pod.ComposeFiles)
	return nil
}

func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir
	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				log.Errorln(err)
				return err
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				log.Errorln(err)
				return err
			}
		}

	}
	return nil
}

func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}
		log.Printf("Copy file from %s to %s\n", sourcefile.Name(), destfile.Name())
	}

	return
}

// Convert array like a=b to map a:b
func ConvertArrayToMap(arr []interface{}) map[interface{}]interface{} {
	m := make(map[interface{}]interface{})
	for _, i := range arr {
		b := strings.SplitN(i.(string), "=", 2)
		if len(b) == 2 {
			m[b[0]] = b[1]
		} else {
			m[b[0]] = ""
		}
	}
	return m
}
