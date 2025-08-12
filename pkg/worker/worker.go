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
	"encoding/json"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	libconfiguration "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	lib_model "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
)

func New(config configuration.Config, libConfig libconfiguration.Config, auth *auth.Auth, smartServiceRepo SmartServiceRepo, w *watcher.Watcher) (*Worker, error) {
	minWatchInterval, err := time.ParseDuration(config.MinWatchInterval)
	if err != nil {
		return nil, err
	}
	defaultWatchInterval, err := time.ParseDuration(config.DefaultWatchInterval)
	if err != nil {
		return nil, err
	}
	return &Worker{
		config:               config,
		libConfig:            libConfig,
		auth:                 auth,
		smartServiceRepo:     smartServiceRepo,
		watcher:              w,
		defaultWatchInterval: defaultWatchInterval,
		minWatchInterval:     minWatchInterval,
	}, nil
}

type Worker struct {
	config               configuration.Config
	libConfig            libconfiguration.Config
	auth                 *auth.Auth
	smartServiceRepo     SmartServiceRepo
	watcher              *watcher.Watcher
	defaultWatchInterval time.Duration
	minWatchInterval     time.Duration
}

type SmartServiceRepo interface {
	GetInstanceUser(instanceId string) (userId string, err error)
	UseModuleDeleteInfo(info lib_model.ModuleDeleteInfo) error
	ListExistingModules(processInstanceId string, query lib_model.ModulQuery) (result []lib_model.SmartServiceModule, err error)
	GetSmartServiceInstance(processInstanceId string) (result lib_model.SmartServiceInstance, err error)
}

func (this *Worker) Do(task lib_model.CamundaExternalTask) (modules []lib_model.Module, outputs map[string]interface{}, err error) {
	sm, err := this.smartServiceRepo.GetSmartServiceInstance(task.ProcessInstanceId)
	if err != nil {
		this.libConfig.GetLogger().Error("ERROR: unable to get instance", "error", err)
		return modules, outputs, err
	}

	id := this.getModuleId(task)
	procedure := this.getMaintenanceProcedureEventName(task)
	httpWatch, err := this.selectWatchedHttpRequest(task)
	if err != nil {
		this.libConfig.GetLogger().Error("ERROR: unable to select watched http request parameter", "error", err)
		return modules, outputs, err
	}

	maintenanceProcedureInputs, err := json.Marshal(this.getMaintenanceProcedureInputs(task))
	if err != nil {
		this.libConfig.GetLogger().Error("ERROR: unable to marshal trigger payload", "error", err)
		return modules, outputs, err
	}

	err = this.watcher.Set(model.WatchedEntityInit{
		Id:       id,
		UserId:   sm.UserId,
		Interval: this.getWatchInterval(task).String(),
		HashType: this.getHashType(task),
		Watch:    httpWatch,
		Trigger: model.HttpRequest{
			Method:       "POST",
			Endpoint:     this.libConfig.SmartServiceRepositoryUrl + "/instances/" + url.PathEscape(sm.Id) + "/maintenance-procedures/" + url.PathEscape(procedure) + "/start",
			Body:         maintenanceProcedureInputs,
			AddAuthToken: true,
		},
		CreatedAt: time.Now().Unix(),
	})

	if err != nil {
		this.libConfig.GetLogger().Error("ERROR: unable to create watcher", "error", err)
		return modules, outputs, err
	}

	moduleDeleteInfo := &lib_model.ModuleDeleteInfo{
		Url:    this.config.AdvertisedUrl + "/watcher/" + url.PathEscape(id),
		UserId: sm.UserId,
	}

	outputs = map[string]interface{}{
		"watcher_id": id,
	}

	modules = []lib_model.Module{{
		Id:               id,
		ProcesInstanceId: task.ProcessInstanceId,
		SmartServiceModuleInit: lib_model.SmartServiceModuleInit{
			DeleteInfo: moduleDeleteInfo,
			ModuleType: this.libConfig.CamundaWorkerTopic,
			ModuleData: outputs,
		},
	}}

	return modules, outputs, err
}

func (this *Worker) Undo(modules []lib_model.Module, reason error) {
	this.libConfig.GetLogger().Error("undo", "reason", reason)
	for _, module := range modules {
		if module.DeleteInfo != nil {
			if module.ModuleType == this.libConfig.CamundaWorkerTopic {
				err := this.watcher.DeleteWatcher(module.DeleteInfo.UserId, module.Id)
				if err != nil {
					this.libConfig.GetLogger().Error("ERROR: unable to delete watcher", "error", err, "stack", string(debug.Stack()))
				}
			} else {
				// keep this code, in case additional moules are added later
				err := this.smartServiceRepo.UseModuleDeleteInfo(*module.DeleteInfo)
				if err != nil {
					this.libConfig.GetLogger().Error("ERROR: unable to use module delete info", "error", err, "stack", string(debug.Stack()))
				}
			}
		}
	}
}

func (this *Worker) getModuleId(task lib_model.CamundaExternalTask) string {
	return task.ProcessInstanceId + "." + task.Id
}
