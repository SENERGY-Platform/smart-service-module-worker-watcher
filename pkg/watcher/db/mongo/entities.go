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

package mongo

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/watcher/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var WatchedEntityBson = getBsonFieldObject[model.WatchedEntity]()

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoCollectionWatchedEntity)
		err = db.ensureCompoundIndex(collection, "entity_id_user_index", true, true, WatchedEntityBson.Id, WatchedEntityBson.UserId)
		if err != nil {
			debug.PrintStack()
			return err
		}
		err = db.ensureIndex(collection, "entity_timestamp_of_next_check_index", "timestamp_of_next_check", true, false)
		if err != nil {
			debug.PrintStack()
			return err
		}
		return nil
	})
}

func (this *Mongo) entityCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoCollectionWatchedEntity)
}

func (this *Mongo) Fetch(max int64) (result []model.WatchedEntity, err error) {
	collection := this.entityCollection()
	opt := options.Find().SetLimit(max)
	err = this.transaction(func(ctx context.Context) (interface{}, error) {
		c, err := collection.Find(ctx, bson.M{"timestamp_of_next_check": bson.M{"$lt": time.Now().Unix()}}, opt)
		if err != nil {
			return nil, err
		}
		result, err = readCursorResult[model.WatchedEntity](ctx, c)
		if err != nil {
			return nil, err
		}
		for i, element := range result {
			dur, err := time.ParseDuration(element.Interval)
			if err != nil {
				this.config.GetLogger().Warn("WARNING: invalid interval in WatchedEntity --> interpret interval as 1 hour", "elementId", element.Id, "elementInterval", element.Interval, "error", err)
				dur = time.Hour
				err = nil
			}
			element.TimestampOfNextCheck = time.Now().Add(dur).Unix()
			result[i] = element
			_, err = collection.UpdateOne(ctx, bson.M{
				WatchedEntityBson.Id:     element.Id,
				WatchedEntityBson.UserId: element.UserId,
			}, bson.M{
				"$set": bson.M{"timestamp_of_next_check": element.TimestampOfNextCheck},
			})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return result, err
}

func (this *Mongo) transaction(f func(ctx context.Context) (interface{}, error)) error {
	if this.config.MongoUseRelSet {
		wc := writeconcern.New(writeconcern.WMajority())
		rc := readconcern.Snapshot()
		txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

		session, err := this.client.StartSession()
		if err != nil {
			return err
		}
		defer session.EndSession(context.Background())

		_, err = session.WithTransaction(context.Background(), func(sessCtx mongo.SessionContext) (interface{}, error) {
			return f(sessCtx)
		}, txnOpts)
		return err
	} else {
		ctx, _ := context.WithTimeout(context.Background(), time.Minute)
		_, err := f(ctx)
		return err
	}
}

func (this *Mongo) UpdateHash(id string, userId string, hash string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.entityCollection().UpdateOne(ctx, bson.M{
		WatchedEntityBson.Id:     id,
		WatchedEntityBson.UserId: userId,
	}, bson.M{
		"$set": bson.M{WatchedEntityBson.LastHash: hash},
	})
	return err
}

func (this *Mongo) Set(element model.WatchedEntityInit) error {
	if element.CreatedAt == 0 {
		element.CreatedAt = time.Now().Unix()
	}
	ctx, _ := getTimeoutContext()
	_, err := this.entityCollection().ReplaceOne(
		ctx,
		bson.M{
			WatchedEntityBson.Id:     element.Id,
			WatchedEntityBson.UserId: element.UserId,
		},
		model.WatchedEntity{
			WatchedEntityInit: element,
			WatchedEntityFetchInfo: model.WatchedEntityFetchInfo{
				TimestampOfNextCheck: 0,
				LastHash:             "",
			},
		},
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) Read(id string, userId string) (result model.WatchedEntity, err error) {
	ctx, _ := getTimeoutContext()
	temp := this.entityCollection().FindOne(ctx, bson.M{WatchedEntityBson.Id: id, WatchedEntityBson.UserId: userId})
	err = temp.Err()
	if err != nil {
		return result, err
	}
	err = temp.Decode(&result)
	return result, err
}

func (this *Mongo) Delete(id string, userId string) error {
	ctx, _ := getTimeoutContext()
	_, err := this.entityCollection().DeleteMany(ctx, bson.M{
		WatchedEntityBson.Id:     id,
		WatchedEntityBson.UserId: userId,
	})
	return err
}

func (this *Mongo) List(filter bson.M, query QueryOptions) (result []model.WatchedEntity, err error) {
	opt := createFindOptions(query)
	ctx, _ := getTimeoutContext()
	if filter == nil {
		filter = bson.M{}
	}
	cursor, err := this.entityCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	return readCursorResult[model.WatchedEntity](ctx, cursor)
}
