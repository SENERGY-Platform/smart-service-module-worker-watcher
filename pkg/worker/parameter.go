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
	"errors"
	"fmt"
	lib_model "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"strings"
	"time"
)

func (this *Worker) getMaintenanceProcedureEventName(task lib_model.CamundaExternalTask) string {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"maintenance_procedure"]
	if !ok {
		return ""
	}
	result, ok := variable.Value.(string)
	if !ok {
		return ""
	}
	return result
}

func (this *Worker) getMaintenanceProcedureInputs(task lib_model.CamundaExternalTask) (result model.SmartServiceParameters) {
	result = model.SmartServiceParameters{}
	for key, variable := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix+"maintenance_procedure_inputs.") {
			id := strings.TrimPrefix(key, this.config.WorkerParamPrefix+"maintenance_procedure_inputs.")
			parameter := model.SmartServiceParameter{
				Id:         id,
				Label:      id,
				ValueLabel: fmt.Sprint(variable.Value),
			}
			switch v := variable.Value.(type) {
			case []byte:
				err := json.Unmarshal(v, &parameter.Value)
				if err != nil {
					parameter.Value = v
				}
			case string:
				err := json.Unmarshal([]byte(v), &parameter.Value)
				if err != nil {
					parameter.Value = v
				}
			default:
				parameter.Value = v
			}
			result = append(result, parameter)
		}
	}
	return result
}

func (this *Worker) getWatchInterval(task lib_model.CamundaExternalTask) (result time.Duration) {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"watch_interval"]
	if !ok {
		return this.defaultWatchInterval
	}
	str, ok := variable.Value.(string)
	if !ok {
		return this.defaultWatchInterval
	}
	result, err := time.ParseDuration(str)
	if err != nil {
		return this.defaultWatchInterval
	}
	if result < this.minWatchInterval {
		return this.minWatchInterval
	}
	return result
}

func (this *Worker) getHashType(task lib_model.CamundaExternalTask) string {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"watch_interval"]
	if !ok {
		return this.config.DefaultHashType
	}
	str, ok := variable.Value.(string)
	if !ok {
		return this.config.DefaultHashType
	}
	return str
}

func (this *Worker) getWatchedHttpRequest(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error) {
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"watch_request"]
	if !ok {
		return req, errors.New("missing watch_request")
	}
	str, ok := variable.Value.(string)
	if !ok {
		return req, errors.New("expect watch_request as json encoded string")
	}
	err = json.Unmarshal([]byte(str), &req)
	if err != nil {
		return req, err
	}
	if req.Header == nil {
		req.Header = map[string][]string{}
	}
	return req, nil
}
