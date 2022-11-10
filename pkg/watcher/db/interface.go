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

package db

import "github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"

type Database interface {
	Fetch(max int64) ([]model.WatchedEntity, error)
	UpdateHash(id string, userId string, hash string) error

	Set(model.WatchedEntityInit) error
	Read(id string, userId string) (model.WatchedEntity, error)
	Delete(id string, userId string) error
}
