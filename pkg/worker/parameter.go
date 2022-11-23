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

func (this *Worker) getMaintenanceProcedureInputs(task lib_model.CamundaExternalTask) (result SmartServiceParameters) {
	result = SmartServiceParameters{}
	for key, variable := range task.Variables {
		if strings.HasPrefix(key, this.config.WorkerParamPrefix+"maintenance_procedure_inputs.") {
			id := strings.TrimPrefix(key, this.config.WorkerParamPrefix+"maintenance_procedure_inputs.")
			parameter := SmartServiceParameter{
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
	variable, ok := task.Variables[this.config.WorkerParamPrefix+"hash_type"]
	if !ok {
		return this.config.DefaultHashType
	}
	str, ok := variable.Value.(string)
	if !ok {
		return this.config.DefaultHashType
	}
	return str
}

func (this *Worker) selectWatchedHttpRequest(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error) {
	selectables := []func(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error){
		this.getWatchedHttpRequest,
		this.getWatchedDevicesHttpRequest,
		this.getWatchedModifiedDevicesHttpRequest,
	}
	for _, f := range selectables {
		req, err := f(task)
		if err == nil {
			return req, err
		}
		if !errors.Is(err, MissingVariableUsage) {
			return req, err
		}
	}
	return req, MissingVariableUsage
}

var MissingVariableUsage = errors.New("missing variable")

func (this *Worker) getWatchedHttpRequest(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error) {
	varName := this.config.WorkerParamPrefix + "watch_request"
	variable, ok := task.Variables[varName]
	if !ok {
		return req, fmt.Errorf("%w: %v", MissingVariableUsage, varName)
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

func (this *Worker) getWatchedDevicesHttpRequest(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error) {
	varName := this.config.WorkerParamPrefix + "watch_devices_by_criteria"
	variable, ok := task.Variables[varName]
	if !ok {
		return req, fmt.Errorf("%w: %v", MissingVariableUsage, varName)
	}
	str, ok := variable.Value.(string)
	if !ok {
		return req, errors.New("expect watch_request as json encoded string")
	}
	criteria := []Criteria{}
	err = json.Unmarshal([]byte(str), &criteria)
	if err != nil {
		return req, err
	}
	req = model.HttpRequest{
		Method:       "POST",
		Endpoint:     this.config.DeviceSelectionApi + "/v2/query/selectables?include_devices=true",
		Body:         []byte(str),
		AddAuthToken: true,
	}
	return req, nil
}

func (this *Worker) getWatchedModifiedDevicesHttpRequest(task lib_model.CamundaExternalTask) (req model.HttpRequest, err error) {
	varName := this.config.WorkerParamPrefix + "watch_modified_devices_by_criteria"
	variable, ok := task.Variables[varName]
	if !ok {
		return req, fmt.Errorf("%w: %v", MissingVariableUsage, varName)
	}
	str, ok := variable.Value.(string)
	if !ok {
		return req, errors.New("expect watch_request as json encoded string")
	}
	criteria := []Criteria{}
	err = json.Unmarshal([]byte(str), &criteria)
	if err != nil {
		return req, err
	}
	req = model.HttpRequest{
		Method:       "POST",
		Endpoint:     this.config.DeviceSelectionApi + "/v2/query/selectables?include_devices=true&include_id_modified=true",
		Body:         []byte(str),
		AddAuthToken: true,
	}
	return req, nil
}
