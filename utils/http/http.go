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

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// generate body for http request
// parse interface{} to io.Reader
func GenBody(t interface{}) io.Reader {
	tjson, err := json.Marshal(&t)
	if err != nil {
		log.Panic("Error marshalling : ", err.Error())
	}
	fmt.Println("Request Body : ", string(tjson))
	return bytes.NewReader(tjson)
}

// http post
func PostRequest(url string, body io.Reader) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Println("Error creating http request : ", err.Error())
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error posting http request : ", err.Error())
		return nil, err
	}
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading http response : ", err.Error())
		return nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		log.Errorf("Failure to close response body :%v", err)
		return nil, err
	}
	return resp_body, nil
}

// http get
func GetRequest(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error creating http request : ", err.Error())
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error getting http request : ", err.Error())
		return nil, err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading http response : ", err.Error())
		return nil, err
	}
	err = resp.Body.Close()
	err = resp.Body.Close()
	if err != nil {
		log.Errorf("Failed to close response body :%v", err)
		return nil, err
	}
	return respBody, nil
}
