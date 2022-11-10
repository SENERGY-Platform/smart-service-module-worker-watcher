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
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"log"
	"sync"
	"time"
)

type Watcher struct {
	config  configuration.Config
	db      db.Database
	checker Checker
	trigger Trigger
}

type Checker interface {
	Check(userId string, request model.HttpRequest, hashType string, lastHash string) (changed bool, newHash string, err error)
}

type Trigger interface {
	Run(userId string, trigger model.HttpRequest) error
}

func New(config configuration.Config, db db.Database, check Checker, trigger Trigger) *Watcher {
	return &Watcher{
		config:  config,
		db:      db,
		checker: check,
		trigger: trigger,
	}
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
				err := this.RunLoop(ctx, this.config.BatchSize)
				if err != nil {
					log.Println("ERROR: Watcher::StartWithInterval::Run()", err)
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
			count, err := this.Run(batchSize)
			if err != nil {
				return err
			}
			if count == 0 {
				return nil
			}
		}
	}
}

func (this *Watcher) Run(batchSize int64) (count int, err error) {
	list, err := this.db.Fetch(batchSize)
	if err != nil {
		return 0, err
	}
	wg := sync.WaitGroup{}
	for _, entity := range list {
		wg.Add(1)
		go func(entity model.WatchedEntity) {
			defer wg.Done()
			chenged, newHash, temperr := this.checker.Check(entity.UserId, entity.Watch, entity.HashType, entity.LastHash)
			if temperr != nil {
				err = temperr
				return
			}
			if chenged {
				temperr = this.db.UpdateHash(entity.Id, entity.UserId, newHash)
				if temperr != nil {
					err = temperr
					return
				}
				temperr = this.trigger.Run(entity.UserId, entity.Trigger)
				if temperr != nil {
					err = temperr
					return
				}
			}
		}(entity)
	}
	wg.Wait()
	return len(list), err
}

func (this *Watcher) DeleteWatcher(userId string, watcherId string) (err error) {
	return this.db.Delete(watcherId, userId)
}
