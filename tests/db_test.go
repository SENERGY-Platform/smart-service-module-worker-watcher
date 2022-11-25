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

package tests

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/db/mongo"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/tests/docker"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestDb(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoUrl, err := docker.MongoRs(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	config := configuration.Config{
		MongoUrl:                     mongoUrl,
		MongoTable:                   "test",
		MongoCollectionWatchedEntity: "test",
		MongoUseRelSet:               true,
	}

	m, err := mongo.New(config, ctx)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("create entities", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			err = m.Set(model.WatchedEntityInit{
				Id:       strconv.Itoa(i),
				UserId:   "user",
				Interval: "1h",
			})
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	done := map[string]bool{}

	t.Run("fetch 10 (1)", func(t *testing.T) {
		fetched, err := m.Fetch(10)
		if err != nil {
			t.Error(err)
			return
		}
		t.Run("check fetched", func(t *testing.T) {
			if len(fetched) != 10 {
				t.Error(len(fetched), fetched)
				return
			}
			for _, f := range fetched {
				if done[f.Id] {
					t.Error("duplicate fetch", f)
					return
				}
				done[f.Id] = true
				if f.TimestampOfNextCheck <= time.Now().Unix() {
					t.Error("unexpected TimestampOfNextCheck", f.TimestampOfNextCheck)
					return
				}
			}
		})
		t.Run("check list", func(t *testing.T) {
			list, err := m.List(nil, MockQueryOptions{sort: "id.asc", limit: 100, offset: 0})
			if err != nil {
				t.Error(err)
				return
			}

			if len(list) != 50 {
				t.Error(len(list), list)
				return
			}

			countOfUnprocessed := 0
			countOfProcessed := 0
			expectedProcessedCount := 10
			expectedUnprocessedCount := 40

			for _, e := range list {
				isFetched := done[e.Id]
				hasSmallTimestamp := e.TimestampOfNextCheck < time.Now().Unix()
				if hasSmallTimestamp {
					countOfUnprocessed = countOfUnprocessed + 1
				} else {
					countOfProcessed = countOfProcessed + 1
				}
				if isFetched && hasSmallTimestamp {
					t.Error("unexpected TimestampOfNextCheck", e.TimestampOfNextCheck)
				}
				if !isFetched && !hasSmallTimestamp {
					t.Error("unexpected TimestampOfNextCheck", e.TimestampOfNextCheck)
				}
			}
			if countOfProcessed != expectedProcessedCount {
				t.Error("unexpected countOfProcessed", countOfProcessed, expectedProcessedCount)
			}
			if countOfUnprocessed != expectedUnprocessedCount {
				t.Error("unexpected countOfUnprocessed", countOfUnprocessed, expectedUnprocessedCount)
			}
		})
	})

	t.Run("fetch 10 (2)", func(t *testing.T) {
		fetched, err := m.Fetch(10)
		if err != nil {
			t.Error(err)
			return
		}
		t.Run("check fetched", func(t *testing.T) {
			if len(fetched) != 10 {
				t.Error(len(fetched), fetched)
				return
			}
			for _, f := range fetched {
				if done[f.Id] {
					t.Error("duplicate fetch", f)
					return
				}
				done[f.Id] = true
				if f.TimestampOfNextCheck <= time.Now().Unix() {
					t.Error("unexpected TimestampOfNextCheck", f.TimestampOfNextCheck)
					return
				}
			}
		})
		t.Run("check list", func(t *testing.T) {
			list, err := m.List(nil, MockQueryOptions{sort: "id.asc", limit: 100, offset: 0})
			if err != nil {
				t.Error(err)
				return
			}

			if len(list) != 50 {
				t.Error(len(list), list)
				return
			}

			countOfUnprocessed := 0
			countOfProcessed := 0
			expectedProcessedCount := 20
			expectedUnprocessedCount := 30

			for _, e := range list {
				isFetched := done[e.Id]
				hasSmallTimestamp := e.TimestampOfNextCheck < time.Now().Unix()
				if hasSmallTimestamp {
					countOfUnprocessed = countOfUnprocessed + 1
				} else {
					countOfProcessed = countOfProcessed + 1
				}
				if isFetched && hasSmallTimestamp {
					t.Error("unexpected TimestampOfNextCheck", e.TimestampOfNextCheck)
				}
				if !isFetched && !hasSmallTimestamp {
					t.Error("unexpected TimestampOfNextCheck", e.TimestampOfNextCheck)
				}
			}
			if countOfProcessed != expectedProcessedCount {
				t.Error("unexpected countOfProcessed", countOfProcessed, expectedProcessedCount)
			}
			if countOfUnprocessed != expectedUnprocessedCount {
				t.Error("unexpected countOfUnprocessed", countOfUnprocessed, expectedUnprocessedCount)
			}
		})
	})

	t.Run("set hash", func(t *testing.T) {
		err = m.UpdateHash("2", "user", "foobar")
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("read updated watcher", func(t *testing.T) {
		watcher, err := m.Read("2", "user")
		if err != nil {
			t.Error(err)
			return
		}
		if watcher.LastHash != "foobar" || watcher.Id != "2" {
			t.Error(watcher)
		}
	})

	t.Run("read unchanged watcher", func(t *testing.T) {
		watcher, err := m.Read("3", "user")
		if err != nil {
			t.Error(err)
			return
		}
		if watcher.LastHash != "" || watcher.Id != "3" {
			t.Error(watcher)
		}
	})

	t.Run("delete watcher", func(t *testing.T) {
		err = m.Delete("4", "user")
		if err != nil {
			t.Error(err)
			return
		}
		list, err := m.List(nil, MockQueryOptions{sort: "id.asc", limit: 100, offset: 0})
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 49 {
			t.Error(len(list), list)
		}
	})

	t.Run("delete idempotent", func(t *testing.T) {
		err = m.Delete("4", "user")
		if err != nil {
			t.Error(err)
			return
		}
		list, err := m.List(nil, MockQueryOptions{sort: "id.asc", limit: 100, offset: 0})
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 49 {
			t.Error(len(list), list)
		}
	})
}

type MockQueryOptions struct {
	sort   string
	limit  int64
	offset int64
}

func (this MockQueryOptions) GetLimit() int64 {
	return this.limit
}

func (this MockQueryOptions) GetOffset() int64 {
	return this.offset
}

func (this MockQueryOptions) GetSort() string {
	return this.sort
}
