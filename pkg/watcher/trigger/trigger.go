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
	libconfig "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/auth"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"io"
	"net/http"
	"time"
)

type Trigger struct {
	config        configuration.Config
	tokenprovider func(userid string) (token auth.Token, err error)
}

func New(config configuration.Config, libConf libconfig.Config) (*Trigger, error) {
	return &Trigger{config: config, tokenprovider: auth.GetCachedTokenProvider(libConf)}, nil
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
		token, err := this.tokenprovider(userId)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", token.Jwt())
	}
	client := http.Client{
		Timeout: 5 * time.Second,
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
