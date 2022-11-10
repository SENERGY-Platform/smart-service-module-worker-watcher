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

package api

import (
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/auth"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &WatcherEndpoints{})
}

type WatcherEndpoints struct{}

// Delete godoc
// @Summary      removes a watcher
// @Description  removes a watcher
// @Tags         watcher
// @Param        id path string true "Watcher ID"
// @Success      200
// @Failure      500
// @Router       /watcher/{id} [delete]
func (this *WatcherEndpoints) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/watcher/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}

		err = ctrl.DeleteWatcher(token.GetUserId(), id)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}
