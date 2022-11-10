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

package model

import (
	"net/http"
)

type WatchedEntity struct {
	WatchedEntityInit      `bson:",inline"`
	WatchedEntityFetchInfo `bson:",inline"`
}

type WatchedEntityInit struct {
	Id       string      `json:"id"`
	UserId   string      `json:"user_id"`
	Interval string      `json:"interval"`
	HashType string      `json:"hash_type"`
	Watch    HttpRequest `json:"watch"`
	Trigger  HttpRequest `json:"trigger"`
}

type WatchedEntityFetchInfo struct {
	TimestampOfNextCheck int64  `json:"timestamp_of_next_check" bson:"timestamp_of_next_check"`
	LastHash             string `json:"last_hash"`
}

type HttpRequest struct {
	Method       string      `json:"method"`
	Endpoint     string      `json:"endpoint"`
	Body         []byte      `json:"body"`
	AddAuthToken bool        `json:"add_auth_token"`
	Header       http.Header `json:"header"`
}
