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
	lib "github.com/SENERGY-Platform/smart-service-module-worker-lib"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	libconfig "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/smartservicerepository"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/checker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/cleanup"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db/mongo"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/trigger"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/worker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/docker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/mocks"
	"os"
	"sync"
	"testing"
	"time"
)

const TEST_CASE_DIR = "./testcases/"

func TestWithMocks(t *testing.T) {
	libConf, err := libconfig.LoadLibConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	conf, err := libconfig.Load[configuration.Config]("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	libConf.CamundaWorkerWaitDurationInMs = 200
	conf.WatchInterval = "1h"
	conf.DeviceSelectionApi = "http://device-selection-url:8080"

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoUrl, err := docker.MongoRs(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	conf.MongoUrl = mongoUrl

	infos, err := os.ReadDir(TEST_CASE_DIR)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() && isValidaForMockTest(TEST_CASE_DIR+name) {
			t.Run(name, func(t *testing.T) {
				runTest(t, TEST_CASE_DIR+name, conf, libConf)
			})
		}
	}
}

func isValidaForMockTest(dir string) bool {
	expectedFiles := []string{
		"camunda_tasks.json",
		"expected_database_requests.json",
		"expected_smart_service_repo_requests.json",
	}
	infos, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	files := map[string]bool{}
	for _, info := range infos {
		if !info.IsDir() {
			files[info.Name()] = true
		}
	}
	for _, expected := range expectedFiles {
		if !files[expected] {
			return false
		}
	}
	return true
}

func runTest(t *testing.T, testCaseLocation string, config configuration.Config, libConf libconfig.Config) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	camunda := mocks.NewCamundaMock()
	libConf.CamundaUrl = camunda.Start(ctx, wg)
	err := camunda.AddFileToQueue(testCaseLocation + "/camunda_tasks.json")
	if err != nil {
		t.Error(err)
		return
	}

	libConf.AuthEndpoint = mocks.Keycloak(ctx, wg)

	smartServiceRepo := mocks.NewSmartServiceRepoMock(libConf, config, []byte{})
	libConf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	m, err := mongo.New(config, ctx)
	if err != nil {
		t.Error(err)
		return
	}
	database := mocks.NewDbRecorder(config, libConf, m)

	err = StartMock(ctx, wg, config, libConf, database)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	err = smartServiceRepo.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/expected_smart_service_repo_requests.json")
	if err != nil {
		t.Error("/expected_smart_service_repo_requests.json", err)
	}

	err = database.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/expected_database_requests.json")
	if err != nil {
		t.Error("/expected_database_requests.json", err)
	}
}

func StartMock(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, libConfig libconfig.Config, db db.Database) error {
	handlerFactory := func(a *auth.Auth, smartServiceRepo *smartservicerepository.SmartServiceRepository) (camunda.Handler, error) {
		c, err := checker.New(a)
		if err != nil {
			return nil, err
		}
		t, err := trigger.New(a)
		if err != nil {
			return nil, err
		}
		cleanupChecker := cleanup.New(smartServiceRepo)
		w := watcher.New(config, db, c, t, cleanupChecker)
		err = w.Start(ctx, wg)
		if err != nil {
			return nil, err
		}
		return worker.New(config, libConfig, a, smartServiceRepo, w)
	}
	return lib.Start(ctx, wg, libConfig, handlerFactory)
}
