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
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/trigger"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/mocks"
	"reflect"
	"sync"
	"testing"
)

func TestTrigger(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetUrl, requestListMux, requstList := mocks.StartTestHttpMock(ctx, wg, []mocks.HttpMockResponse{
		{Code: 200, Payload: []byte("foobar123")},
		{Code: 200},
	})

	expectedRequests := []model.HttpRequest{
		{
			Method:   "POST",
			Endpoint: "/query",
			Body:     []byte(`{"foo":"bar"}`),
			Header:   map[string][]string{"Authorization": {"test-user"}},
		},
		{
			Method:   "POST",
			Endpoint: "/query",
			Body:     []byte(`{"foo":"bar"}`),
			Header:   map[string][]string{},
		},
	}

	tr, err := trigger.New(mocks.AuthMock{})
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("trigger with auth", func(t *testing.T) {
		err = tr.Run("test-user", model.HttpRequest{
			Method:       "POST",
			Endpoint:     targetUrl + "/query",
			Body:         []byte(`{"foo":"bar"}`),
			AddAuthToken: true,
			Header:       nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("trigger without auth", func(t *testing.T) {
		err = tr.Run("test-user", model.HttpRequest{
			Method:       "POST",
			Endpoint:     targetUrl + "/query",
			Body:         []byte(`{"foo":"bar"}`),
			AddAuthToken: false,
			Header:       nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check requests", func(t *testing.T) {
		requestListMux.Lock()
		defer requestListMux.Unlock()
		cleanedActualRequests := MapList(*requstList, func(in model.HttpRequest) model.HttpRequest {
			atoken := in.Header.Get("Authorization")
			if atoken != "" {
				token, err := auth.Parse(atoken)
				if err != nil {
					t.Error(err)
				}
				in.Header = map[string][]string{"Authorization": {token.GetUserId()}}
			} else {
				in.Header = map[string][]string{}
			}

			return in
		})
		if !reflect.DeepEqual(expectedRequests, cleanedActualRequests) {
			actualJson, _ := json.Marshal(cleanedActualRequests)
			expectedJson, _ := json.Marshal(expectedRequests)
			t.Error("\n", string(expectedJson), "\n", string(actualJson))
		}
	})
}
