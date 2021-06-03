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
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/paypal/dce-go/config"
	log "github.com/sirupsen/logrus"
)

// GenBody : generate body for http request, convert body from interface{}  to io.Reader
func GenBody(body interface{}) io.Reader {
	jsonBody, err := json.Marshal(&body)
	if err != nil {
		log.Panic("error marshalling : ", err.Error())
	}
	log.Infof("request body : %s", string(jsonBody))
	return bytes.NewReader(jsonBody)
}

// PostRequest : http POST request
func PostRequest(ctx context.Context, transport http.RoundTripper, url string, body io.Reader) ([]byte, error) {
	headers := map[string]string{"Content-Type": "application/json"}
	return getHttpResponse(ctx, transport, url, "POST", headers, body)
}

// GetRequest :  http GET request
func GetRequest(ctx context.Context, transport http.RoundTripper, url string) ([]byte, error) {
	headers := map[string]string{"Accept": "application/json"}
	return getHttpResponse(ctx, transport, url, "GET", headers, nil)
}

func getHttpResponse(ctx context.Context, transport http.RoundTripper, url, methodType string, headers map[string]string,
	body io.Reader) ([]byte, error) {
	
	client := &http.Client{
		Transport: getTransportValue(transport),
		Timeout:   config.GetHttpTimeout(),
	}

	req, err := http.NewRequestWithContext(ctx, methodType, url, body)
	if err != nil {
		log.Errorf("error creating http %s request for: %s", methodType, err.Error())
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("http %s request failed: %s", methodType, err.Error())
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("failed to close response body: %s", err.Error())
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error reading http response : %s", err.Error())
		return nil, err
	}

	return respBody, nil
}

// returns the http.DefaultTransport if transport is nil
func getTransportValue(transport http.RoundTripper) http.RoundTripper {
	if transport != nil {
		return transport
	}
	return http.DefaultTransport
}
