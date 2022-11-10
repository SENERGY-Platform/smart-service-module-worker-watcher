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

package worker

import (
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	libconfiguration "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"
	"log"
	"runtime/debug"
)

func New(config configuration.Config, libConfig libconfiguration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo, w *watcher.Watcher) *Worker {
	return &Worker{config: config, libConfig: libConfig, auth: auth, smartServiceRepo: smartServiceRepo, watcher: w}
}

type Worker struct {
	config           configuration.Config
	libConfig        libconfiguration.Config
	auth             *auth.Auth
	smartServiceRepo SmartServiceRepo
	watcher          *watcher.Watcher
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
	UseModuleDeleteInfo(info model.ModuleDeleteInfo) error
	ListExistingModules(processInstanceId string, query model.ModulQuery) (result []model.SmartServiceModule, err error)
}

func (this *Worker) Do(task model.CamundaExternalTask) (modules []model.Module, outputs map[string]interface{}, err error) {
	userId, err := this.smartServiceRepo.GetInstanceUser(task.ProcessInstanceId)
	if err != nil {
		log.Println("ERROR: unable to get instance user", err)
		return modules, outputs, err
	}
	token, err := this.auth.ExchangeUserToken(userId)
	if err != nil {
		log.Println("ERROR: unable to exchange user token", err)
		return modules, outputs, err
	}

	//TODO: replace with real code
	log.Println(token)

	return modules, outputs, err
}

func (this *Worker) Undo(modules []model.Module, reason error) {
	log.Println("UNDO:", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil {
			err := this.smartServiceRepo.UseModuleDeleteInfo(*module.DeleteInfo)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
			}
		}
	}
}
