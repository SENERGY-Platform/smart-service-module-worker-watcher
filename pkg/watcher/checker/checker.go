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

package checker

import (
	"bytes"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"io"
	"net/http"
	"time"
)

type Checker struct {
	config configuration.Config
	auth   *auth.Auth
}

func New(config configuration.Config, auth *auth.Auth) (*Checker, error) {
	return &Checker{config: config, auth: auth}, nil
}

func (this *Checker) Check(userId string, request model.HttpRequest, hashType string, lastHash string) (changed bool, newHash string, err error) {
	payload, err := this.request(userId, request)
	if err != nil {
		return false, "", err
	}
	newHash, err = hash(hashType, payload)
	if err != nil {
		return false, "", err
	}
	if lastHash != newHash {
		changed = true
	}
	return changed, newHash, nil
}

func (this *Checker) request(userId string, trigger model.HttpRequest) (payload []byte, err error) {
	req, err := http.NewRequest(trigger.Method, trigger.Endpoint, bytes.NewReader(trigger.Body))
	if err != nil {
		return nil, err
	}
	for key, value := range trigger.Header {
		req.Header[key] = value
	}
	if trigger.AddAuthToken {
		token, err := this.auth.ExchangeUserToken(userId)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", token.Jwt())
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	payload, _ = io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected trigger response: %v, %v", resp.StatusCode, string(payload))
	}
	return payload, nil
}
