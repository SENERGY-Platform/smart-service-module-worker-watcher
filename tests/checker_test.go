/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/checker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/mocks"
	"reflect"
	"sync"
	"testing"
)

func TestChecker(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetUrl, requestListMux, requstList := mocks.StartTestHttpMock(ctx, wg, []mocks.HttpMockResponse{
		{Code: 200, Payload: []byte("foobar123")},
		{Code: 200, Payload: []byte("foobar123")},
		{Code: 200, Payload: []byte("changed")},
		{Code: 200, Payload: []byte("changed")},
	})

	expectedRequest := model.HttpRequest{
		Method:   "POST",
		Endpoint: "/query",
		Body:     []byte(`{"foo":"bar"}`),
		Header:   map[string][]string{"Authorization": {"test-user"}},
	}
	expectedRequests := []model.HttpRequest{expectedRequest, expectedRequest, expectedRequest, expectedRequest}

	c, err := checker.New(configuration.Config{ExternalDnsAddress: "8.8.8.8:53"}, mocks.AuthMock{})
	if err != nil {
		t.Error(err)
		return
	}

	lastHash := ""
	checkRequest := model.HttpRequest{
		Method:       "POST",
		Endpoint:     targetUrl + "/query",
		Body:         []byte(`{"foo":"bar"}`),
		AddAuthToken: true,
		Header:       nil,
	}

	t.Run("isolated local check", func(t *testing.T) {
		_, _, err := c.Check("test-user", model.HttpRequest{
			Method:       "POST",
			Endpoint:     targetUrl + "/query",
			Body:         []byte(`{"foo":"bar"}`),
			AddAuthToken: true,
			Header:       nil,
			Isolated:     true,
		}, checker.HASH_TYPE_MD5, lastHash)
		if err == nil {
			t.Error("missing 'is not a public IP address' error")
			return
		}
	})

	t.Run("isolated public check", func(t *testing.T) {
		_, _, err := c.Check("test-user", model.HttpRequest{
			Method:       "GET",
			Endpoint:     "http://example.com",
			Body:         nil,
			AddAuthToken: false,
			Header:       nil,
			Isolated:     true,
		}, checker.HASH_TYPE_MD5, lastHash)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("first check", func(t *testing.T) {
		expectedNewHash := "ae2d699aca20886f6bed96a0425c6168"
		changed, newHash, err := c.Check("test-user", checkRequest, checker.HASH_TYPE_MD5, lastHash)
		if err != nil {
			t.Error(err)
			return
		}
		if !changed {
			t.Error(changed, newHash, lastHash)
		}
		if newHash != expectedNewHash {
			t.Error(newHash, expectedNewHash)
		}
		lastHash = newHash
	})

	t.Run("unchanged", func(t *testing.T) {
		expectedNewHash := "ae2d699aca20886f6bed96a0425c6168"
		changed, newHash, err := c.Check("test-user", checkRequest, checker.HASH_TYPE_MD5, lastHash)
		if err != nil {
			t.Error(err)
			return
		}
		if changed {
			t.Error(changed, newHash, lastHash)
		}
		if newHash != expectedNewHash {
			t.Error(newHash, expectedNewHash)
		}
		lastHash = newHash
	})

	t.Run("changed", func(t *testing.T) {
		expectedNewHash := "8977dfac2f8e04cb96e66882235f5aba"
		changed, newHash, err := c.Check("test-user", checkRequest, checker.HASH_TYPE_MD5, lastHash)
		if err != nil {
			t.Error(err)
			return
		}
		if !changed {
			t.Error(changed, newHash, lastHash)
		}
		if newHash != expectedNewHash {
			t.Error(newHash, expectedNewHash)
		}
		lastHash = newHash
	})

	t.Run("kept change", func(t *testing.T) {
		expectedNewHash := "8977dfac2f8e04cb96e66882235f5aba"
		changed, newHash, err := c.Check("test-user", checkRequest, checker.HASH_TYPE_MD5, lastHash)
		if err != nil {
			t.Error(err)
			return
		}
		if changed {
			t.Error(changed, newHash, lastHash)
		}
		if newHash != expectedNewHash {
			t.Error(newHash, expectedNewHash)
		}
		lastHash = newHash
	})

	t.Run("check requests", func(t *testing.T) {
		requestListMux.Lock()
		defer requestListMux.Unlock()
		cleanedActualRequests := MapList(*requstList, func(in model.HttpRequest) model.HttpRequest {
			token, err := auth.Parse(in.Header.Get("Authorization"))
			if err != nil {
				t.Error(err)
			}
			in.Header = map[string][]string{"Authorization": {token.GetUserId()}}
			return in
		})
		if !reflect.DeepEqual(expectedRequests, cleanedActualRequests) {
			actualJson, _ := json.Marshal(cleanedActualRequests)
			expectedJson, _ := json.Marshal(expectedRequests)
			t.Error("\n", string(expectedJson), "\n", string(actualJson))
		}
	})
}

func MapList[In any, Out any](list []In, f func(in In) Out) (result []Out) {
	for _, v := range list {
		result = append(result, f(v))
	}
	return result
}
