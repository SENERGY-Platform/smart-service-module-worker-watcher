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

package pkg

import (
	"context"
	lib "github.com/SENERGY-Platform/smart-service-module-worker-lib"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/camunda"
	libconfiguration "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/smartservicerepository"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/checker"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/cleanup"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db/mongo"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/trigger"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/worker"
	"sync"
)

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, libConfig libconfiguration.Config) error {
	handlerFactory := func(a *auth.Auth, smartServiceRepo *smartservicerepository.SmartServiceRepository) (camunda.Handler, error) {
		db, err := mongo.New(config, ctx)
		if err != nil {
			return nil, err
		}
		c, err := checker.New(a)
		if err != nil {
			return nil, err
		}
		t, err := trigger.New(a)
		if err != nil {
			return nil, err
		}
		w := watcher.New(config, db, c, t)
		err = w.Start(ctx, wg)
		if err != nil {
			return nil, err
		}
		err = cleanup.Start(ctx, wg, config, w, smartServiceRepo)
		if err != nil {
			return nil, err
		}
		return worker.New(config, libConfig, a, smartServiceRepo, w), nil
	}
	return lib.Start(ctx, wg, libConfig, handlerFactory)
}
