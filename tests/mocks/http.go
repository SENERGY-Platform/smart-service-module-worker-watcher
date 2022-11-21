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
	"context"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

type HttpMockResponse struct {
	Payload []byte
	Code    int
}

func StartTestHttpMock(ctx context.Context, wg *sync.WaitGroup, responseList []HttpMockResponse) (url string, mux *sync.Mutex, requests *[]model.HttpRequest) {
	requestList := []model.HttpRequest{}
	requests = &requestList
	mux = &sync.Mutex{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mux.Lock()
		defer mux.Unlock()

		respIndex := len(requestList)

		msg := model.HttpRequest{
			Method:   request.Method,
			Endpoint: request.URL.String(),
			Header:   request.Header,
		}
		msg.Body, _ = io.ReadAll(request.Body)
		requestList = append(requestList, msg)

		if len(responseList) > respIndex {
			resp := responseList[respIndex]
			writer.WriteHeader(resp.Code)
			writer.Write(resp.Payload)
		}

	}))
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		<-ctx.Done()
		server.Close()
	}()
	url = server.URL
	return url, mux, requests
}
