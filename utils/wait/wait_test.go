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

package wait

import (
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestPollUntil(t *testing.T) {
	var done chan string
	// timeout
	_, err := PollUntil(1*time.Millisecond, done, 1*time.Millisecond, ConditionFunc(func() (string, error) {
		time.Sleep(20 * time.Second)
		return "hello", nil
	}))
	if err == nil || err != ErrTimeOut {
		t.Fatalf("expected timeout err, but got %v", err)
	}

	//conditionfunc finish
	res, _ := PollUntil(1*time.Second, done, 1*time.Second, ConditionFunc(func() (string, error) {
		return "hello", nil
	}))
	if res != "hello" {
		t.Fatalf("expected return message to be 'hello', but got %s", res)
	}
}

func TestWaitUntil(t *testing.T) {
	//timeout
	_, err := WaitUntil(1*time.Millisecond, ConditionCHFunc(func(reply chan string) {
		time.Sleep(2 * time.Second)
		reply <- "hello"
	}))
	if err == nil || err != ErrTimeOut {
		t.Fatalf("expected timeout err, but got %v", err)
	}

	//conditionfunc finish
	res, _ := WaitUntil(5*time.Second, ConditionCHFunc(func(reply chan string) {
		reply <- "hello"
	}))

	if res != "hello" {
		t.Fatalf("expected return message to be 'hello', but got %s", res)
	}
}

func TestPollRetry(t *testing.T) {
	//retry 3 times and return error
	err := PollRetry(3, 200*time.Millisecond, ConditionFunc(func() (string, error) {
		return "", errors.New("testerror")
	}))
	if err == nil || err != ErrTimeOut {
		t.Fatalf("expected timeout err, but got %v", err)
	}

	//success
	err = PollRetry(3, 200*time.Millisecond, ConditionFunc(func() (string, error) {
		return "", nil
	}))
	if err != nil {
		t.Fatalf("expected no err, but got %v", err)
	}
}
