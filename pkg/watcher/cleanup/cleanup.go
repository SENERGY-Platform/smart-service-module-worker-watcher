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

package cleanup

import (
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/smartservicerepository"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"net/http"
	"time"
)

type Checker struct {
	smr *smartservicerepository.SmartServiceRepository
}

func New(smr *smartservicerepository.SmartServiceRepository) *Checker {
	return &Checker{smr: smr}
}

func (this *Checker) Check(entity model.WatchedEntity) (remove bool, err error) {
	return Check(entity, this.smr)
}

func Check(element model.WatchedEntity, smr *smartservicerepository.SmartServiceRepository) (remove bool, err error) {
	if time.Since(time.Unix(element.CreatedAt, 0)) < time.Minute {
		return false, nil
	}
	_, err, code := smr.GetModule(element.UserId, element.Id)
	if code == http.StatusNotFound {
		return true, nil
	}
	return false, err
}

type QueryOptions struct {
	limit  int64
	offset int64
}

func (this QueryOptions) GetLimit() int64 {
	return this.limit
}

func (this QueryOptions) GetOffset() int64 {
	return this.offset
}

func (this QueryOptions) GetSort() string {
	return "id.asc"
}
