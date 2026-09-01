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

	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"github.com/SENERGY-Platform/service-commons/pkg/accesslog"
	libconfiguration "github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/api/util"
	"github.com/julienschmidt/httprouter"
)

type EndpointMethod = func(config configuration.Config, router *httprouter.Router, ctrl Controller)

var endpoints = []interface{}{} //list of objects with EndpointMethod

type Controller interface {
	DeleteWatcher(ctx context.Context, userId string, watcherId string) (err error)
}

func Start(ctx context.Context, config configuration.Config, libConfig libconfiguration.Config, ctrl Controller) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	router := GetRouter(ctx, config, libConfig, ctrl)

	advertisedUrl, err := url.Parse(config.AdvertisedUrl)
	if err != nil {
		return err
	}

	server := &http.Server{Addr: ":" + advertisedUrl.Port(), Handler: router}
	go func() {
		config.GetLogger().InfoContext(ctx, "listening on "+server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			config.GetLogger().ErrorContext(ctx, "error starting server", "error", err)
			log.Fatal("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		config.GetLogger().InfoContext(ctx, "api shutdown", "error", server.Shutdown(context.Background()))
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
func GetRouter(ctx context.Context, config configuration.Config, libConfig libconfiguration.Config, command Controller) http.Handler {
	router := httprouter.New()
	for _, e := range endpoints {
		for name, call := range getEndpointMethods(e) {
			config.GetLogger().InfoContext(ctx, "add endpoint "+name)
			call(config, router, command)
		}
	}

	var handler http.Handler = router
	//HTTPOpenTelemetry extracts the trace-context of incoming requests; a failure must not
	//keep this server from serving, tracing is best-effort.
	otelHandler, err := otelx.HTTPOpenTelemetry(ctx, libConfig.OtelEndpoint, libconfiguration.ServiceName(), handler)
	if err != nil {
		config.GetLogger().ErrorContext(ctx, "unable to init open-telemetry -> continue without tracing", "error", err)
	} else {
		handler = otelHandler
	}
	return accesslog.New(util.NewCors(handler))
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
