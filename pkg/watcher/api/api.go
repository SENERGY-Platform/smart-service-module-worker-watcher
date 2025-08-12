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
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"

	"github.com/SENERGY-Platform/service-commons/pkg/accesslog"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/api/util"
	"github.com/julienschmidt/httprouter"
)

type EndpointMethod = func(config configuration.Config, router *httprouter.Router, ctrl Controller)

var endpoints = []interface{}{} //list of objects with EndpointMethod

type Controller interface {
	DeleteWatcher(userId string, watcherId string) (err error)
}

func Start(ctx context.Context, config configuration.Config, ctrl Controller) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	router := GetRouter(config, ctrl)

	advertisedUrl, err := url.Parse(config.AdvertisedUrl)
	if err != nil {
		return err
	}

	server := &http.Server{Addr: ":" + advertisedUrl.Port(), Handler: router}
	go func() {
		config.GetLogger().Info("listening on " + server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			config.GetLogger().Error("error starting server", "error", err)
			log.Fatal("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		config.GetLogger().Info("api shutdown", "error", server.Shutdown(context.Background()))
	}()
	return
}

// @title         Smart-Service-Repository API
// @version       0.1
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @host      localhost:8080
// @BasePath  /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func GetRouter(config configuration.Config, command Controller) http.Handler {
	router := httprouter.New()
	for _, e := range endpoints {
		for name, call := range getEndpointMethods(e) {
			config.GetLogger().Info("add endpoint " + name)
			call(config, router, command)
		}
	}
	return accesslog.New(util.NewCors(router))
}

func getEndpointMethods(e interface{}) map[string]func(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	result := map[string]EndpointMethod{}
	objRef := reflect.ValueOf(e)
	methodCount := objRef.NumMethod()
	for i := 0; i < methodCount; i++ {
		m := objRef.Method(i)
		f, ok := m.Interface().(EndpointMethod)
		if ok {
			name := getTypeName(objRef.Type()) + "::" + objRef.Type().Method(i).Name
			result[name] = f
		}
	}
	return result
}

func getTypeName(t reflect.Type) (res string) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
