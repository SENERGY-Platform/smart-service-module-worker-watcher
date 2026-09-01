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

package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracerName identifies this package as the source of the spans it creates.
const TracerName = "github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher"

type Watcher struct {
	config         configuration.Config
	db             db.Database
	checker        Checker
	trigger        Trigger
	cleanupChecker CleanupChecker
}

type Checker interface {
	Check(ctx context.Context, userId string, request model.HttpRequest, hashType string, lastHash string) (changed bool, newHash string, err error)
}

type Trigger interface {
	Run(ctx context.Context, userId string, trigger model.HttpRequest) error
}

type CleanupChecker interface {
	Check(ctx context.Context, entity model.WatchedEntity) (remove bool, err error)
}

func New(config configuration.Config, db db.Database, check Checker, trigger Trigger, cleanupChecker CleanupChecker) *Watcher {
	return &Watcher{
		config:         config,
		db:             db,
		checker:        check,
		trigger:        trigger,
		cleanupChecker: cleanupChecker,
	}
}

func (this *Watcher) Set(ctx context.Context, entity model.WatchedEntityInit) error {
	return this.db.Set(ctx, entity)
}

// Start watching cycle with configured WatchInterval
// wg may be nil
func (this *Watcher) Start(ctx context.Context, wg *sync.WaitGroup) error {
	interval, err := time.ParseDuration(this.config.WatchInterval)
	if err != nil {
		return err
	}
	this.StartWithInterval(ctx, wg, interval)
	return nil
}

// StartWithInterval starts watching cycle with given interval
// wg may be nil
func (this *Watcher) StartWithInterval(ctx context.Context, wg *sync.WaitGroup, interval time.Duration) {
	ticker := time.NewTicker(interval)
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		defer func() {
			ticker.Stop()
			if wg != nil {
				wg.Done()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				//this cycle is not triggered by an incoming request, so ctx carries no trace to
				//inherit; it is the background context this Watcher was started with.
				err := this.RunLoop(ctx, this.config.BatchSize)
				if err != nil {
					this.config.GetLogger().ErrorContext(ctx, "ERROR: Watcher::StartWithInterval::Run()", "error", err)
				}
			}
		}
	}()
}

// RunLoop calls Run until count is 0, an error is returned ore ctx is done
func (this *Watcher) RunLoop(ctx context.Context, batchSize int64) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			count, err := this.Run(ctx, batchSize)
			if err != nil {
				return err
			}
			if count == 0 {
				return nil
			}
		}
	}
}

func (this *Watcher) Run(ctx context.Context, batchSize int64) (count int, err error) {
	list, err := this.db.Fetch(ctx, batchSize)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		//most ticks find nothing to do: a span for every empty poll would be pure noise.
		//only a batch with actual entities to process gets its own span below.
		return 0, nil
	}
	ctx, span := otel.Tracer(TracerName).Start(ctx, "watcher.run", trace.WithAttributes(
		attribute.Int("watcher_batch_size", len(list)),
	))
	defer span.End()
	wg := sync.WaitGroup{}
	for _, entity := range list {
		wg.Add(1)
		go func(entity model.WatchedEntity) {
			defer wg.Done()
			remove, temperr := this.cleanupChecker.Check(ctx, entity)
			if temperr != nil {
				err = temperr
				return
			}
			if remove {
				temperr = this.db.Delete(ctx, entity.Id, entity.UserId)
				if temperr != nil {
					err = temperr
					return
				}
				return
			}
			chenged, newHash, temperr := this.checker.Check(ctx, entity.UserId, entity.Watch, entity.HashType, entity.LastHash)
			if temperr != nil {
				err = temperr
				return
			}
			if chenged {
				temperr = this.db.UpdateHash(ctx, entity.Id, entity.UserId, newHash)
				if temperr != nil {
					err = temperr
					return
				}
				if entity.LastHash != "" {
					temperr = this.trigger.Run(ctx, entity.UserId, entity.Trigger)
					if temperr != nil {
						err = temperr
						return
					}
				}
			}
		}(entity)
	}
	wg.Wait()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return len(list), err
}

func (this *Watcher) DeleteWatcher(ctx context.Context, userId string, watcherId string) (err error) {
	return this.db.Delete(ctx, watcherId, userId)
}
