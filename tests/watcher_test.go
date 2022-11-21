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
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/checker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db/mongo"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/trigger"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/docker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/mocks"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestWatcher(t *testing.T) {

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoUrl, err := docker.MongoRs(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	config := configuration.Config{
		MongoUrl:                     mongoUrl,
		MongoTable:                   "test",
		MongoCollectionWatchedEntity: "test",
		WatchInterval:                "300ms",
		BatchSize:                    10,
	}

	a := mocks.AuthMock{}

	db, err := mongo.New(config, ctx)
	if err != nil {
		t.Error(err)
		return
	}
	c, err := checker.New(a)
	if err != nil {
		t.Error(err)
		return
	}
	tr, err := trigger.New(a)
	if err != nil {
		t.Error(err)
		return
	}
	w := watcher.New(config, db, c, tr, mocks.CleanupChecker{})
	err = w.Start(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	targetUrl, requestListMux, requstList := mocks.StartTestHttpMock(ctx, wg, []mocks.HttpMockResponse{
		{Code: 200, Payload: []byte("foobar123")},
		{Code: 200, Payload: []byte("foobar123")},
		{Code: 200, Payload: []byte("changed")},
		{Code: 200, Payload: []byte("changed")},
		{Code: 200, Payload: []byte("changed")},
		{Code: 200, Payload: []byte("changed")},
	})

	expectetQuery := model.HttpRequest{
		Method:   "GET",
		Endpoint: "/query",
		Body:     []byte{},
		Header:   map[string][]string{},
	}
	expectetTrigger := model.HttpRequest{
		Method:   "POST",
		Endpoint: "/set",
		Body:     []byte(`{"foo":"bar"}`),
		Header:   map[string][]string{},
	}
	expectedRequests := []model.HttpRequest{expectetQuery, expectetQuery, expectetQuery, expectetTrigger, expectetQuery}

	t.Run("add watcher", func(t *testing.T) {
		err = db.Set(model.WatchedEntityInit{
			Id:       "w1",
			UserId:   "test-user",
			Interval: "1s",
			HashType: checker.HASH_TYPE_MD5,
			Watch: model.HttpRequest{
				Method:   "GET",
				Endpoint: targetUrl + "/query",
			},
			Trigger: model.HttpRequest{
				Method:   "POST",
				Endpoint: targetUrl + "/set",
				Body:     []byte(`{"foo":"bar"}`),
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(6300 * time.Millisecond)
	t.Run("stop watcher", func(t *testing.T) {
		err = w.DeleteWatcher("test-user", "w1")
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check requests", func(t *testing.T) {
		requestListMux.Lock()
		defer requestListMux.Unlock()
		cleanedActualRequests := MapList(*requstList, func(in model.HttpRequest) model.HttpRequest {
			in.Header = map[string][]string{}
			return in
		})
		if !reflect.DeepEqual(expectedRequests, cleanedActualRequests) {
			actualJson, _ := json.Marshal(cleanedActualRequests)
			expectedJson, _ := json.Marshal(expectedRequests)
			t.Error("\n", string(expectedJson), "\n", string(actualJson))
		}
	})
}
