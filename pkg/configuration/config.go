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

package configuration

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
)

type Config struct {
	AdvertisedUrl                string `json:"advertised_url"`
	MongoUseRelSet               bool   `json:"mongo_use_rel_set"`
	MongoUrl                     string `json:"mongo_url"`
	MongoTable                   string `json:"mongo_table"`
	MongoCollectionWatchedEntity string `json:"mongo_collection_watched_entity"`
	WatchInterval                string `json:"watch_interval"`
	BatchSize                    int64  `json:"batch_size"`
	WorkerParamPrefix            string `json:"worker_param_prefix"`
	MinWatchInterval             string `json:"min_watch_interval"`
	DefaultWatchInterval         string `json:"default_watch_interval"`
	DefaultHashType              string `json:"default_hash_type"`
	DeviceSelectionUrl           string `json:"device_selection_url"`
	AllowGenericWatchRequests    bool   `json:"allow_generic_watch_requests"`
	UseExternalDnsForChecker     bool   `json:"use_external_dns_for_checker"`
	ExternalDnsAddress           string `json:"external_dns_address"`

	LogLevel string       `json:"log_level"`
	logger   *slog.Logger `json:"-"`
}

func (this *Config) GetLogger() *slog.Logger {
	if this.logger == nil {
		info, ok := debug.ReadBuildInfo()
		project := ""
		if ok {
			if parts := strings.Split(info.Main.Path, "/"); len(parts) > 2 {
				project = strings.Join(parts[2:], "/")
			}
		}
		this.logger = struct_logger.New(
			struct_logger.Config{
				Handler:    struct_logger.JsonHandlerSelector,
				Level:      this.LogLevel,
				TimeFormat: time.RFC3339Nano,
				TimeUtc:    true,
				AddMeta:    true,
			},
			os.Stdout,
			"",
			project).With("project-group", "smart-service")
	}
	return this.logger
}
