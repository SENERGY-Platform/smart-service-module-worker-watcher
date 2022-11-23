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

package trigger

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

type Trigger struct {
	auth           Auth
	client         *http.Client
	isolatedClient *http.Client
}

type Auth interface {
	ExchangeUserToken(userid string) (token auth.Token, err error)
}

func New(config configuration.Config, auth Auth) (*Trigger, error) {
	return &Trigger{
		auth: auth,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		isolatedClient: config.GetSaveHttpClient(),
	}, nil
}

func (this *Trigger) Run(userId string, trigger model.HttpRequest) error {
	req, err := http.NewRequest(trigger.Method, trigger.Endpoint, bytes.NewReader(trigger.Body))
	if err != nil {
		return err
	}
	for key, value := range trigger.Header {
		req.Header[key] = value
	}
	if trigger.AddAuthToken {
		token, err := this.auth.ExchangeUserToken(userId)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", token.Jwt())
	}
	var client *http.Client
	if trigger.Isolated {
		client = this.isolatedClient
	} else {
		client = this.client
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected trigger response: %v, %v", resp.StatusCode, string(temp))
	}
	return nil
}
