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

package mocks

import (
	"encoding/json"
	"fmt"
	libconfig "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"os"
	"reflect"
	"strings"
)

type DbRecorder struct {
	config    configuration.Config
	libconfig libconfig.Config
	db        db.Database
	records   map[string][]map[string]interface{}
}

func NewDbRecorder(config configuration.Config, libconfig libconfig.Config, db db.Database) *DbRecorder {
	return &DbRecorder{
		config:    config,
		libconfig: libconfig,
		db:        db,
		records:   map[string][]map[string]interface{}{},
	}
}

func (this *DbRecorder) Fetch(max int64) ([]model.WatchedEntity, error) {
	this.records["Fetch"] = append(this.records["Fetch"], map[string]interface{}{"max": max})
	return this.db.Fetch(max)
}

func (this *DbRecorder) UpdateHash(id string, userId string, hash string) error {
	this.records["UpdateHash"] = append(this.records["UpdateHash"], map[string]interface{}{"id": id, "userId": userId, "hash": hash})
	return this.db.UpdateHash(id, userId, hash)
}

func (this *DbRecorder) Set(init model.WatchedEntityInit) error {
	init.CreatedAt = 0
	init.Trigger.Endpoint = "http://smr:8080" + strings.TrimPrefix(init.Trigger.Endpoint, this.libconfig.SmartServiceRepositoryUrl)

	this.records["Set"] = append(this.records["Set"], map[string]interface{}{"init": init})
	return this.db.Set(init)
}

func (this *DbRecorder) Read(id string, userId string) (model.WatchedEntity, error) {
	this.records["Read"] = append(this.records["Read"], map[string]interface{}{"id": id, "userId": userId})
	return this.db.Read(id, userId)
}

func (this *DbRecorder) Delete(id string, userId string) error {
	this.records["Delete"] = append(this.records["Delete"], map[string]interface{}{"id": id, "userId": userId})
	return this.db.Delete(id, userId)
}

func (this *DbRecorder) CheckExpectedRequestsFromFileLocation(fileLocation string) error {
	fileContent, err := os.ReadFile(fileLocation)
	if err != nil {
		return err
	}
	expectedRequests := map[string][]map[string]interface{}{}
	err = json.Unmarshal(fileContent, &expectedRequests)
	if err != nil {
		return err
	}
	return this.CheckExpectedRequests(expectedRequests)
}

func (this *DbRecorder) CheckExpectedRequests(expectedRequests map[string][]map[string]interface{}) error {
	actualEngineRequests := this.records
	if !reflect.DeepEqual(homogenize(expectedRequests), homogenize(actualEngineRequests)) {
		a, _ := json.Marshal(actualEngineRequests)
		e, _ := json.Marshal(expectedRequests)
		return fmt.Errorf("\n %v \n %v", string(a), string(e))
	}
	return nil
}

func homogenize(in interface{}) (out interface{}) {
	temp, _ := json.Marshal(in)
	json.Unmarshal(temp, &out)
	return
}
