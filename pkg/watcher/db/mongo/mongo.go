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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-module-worker-watcher/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"reflect"
	"time"
)

type Mongo struct {
	config configuration.Config
	client *mongo.Client
}

var CreateCollections = []func(db *Mongo) error{}

var (
	errEmptyDatabase   = errors.New("mongo database name must not be empty")
	errMissingPassword = errors.New("mongo password must not be empty when a mongo user is set")
)

func New(conf configuration.Config, ctx context.Context) (*Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	return start(ctx, conf, clientOptions(conf), 10*time.Second)
}

// start disconnects the client on every failure path, so a failed startup leaves nothing connected.
func start(ctx context.Context, conf configuration.Config, opts *options.ClientOptions, timeout time.Duration) (*Mongo, error) {
	connectCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client, err := mongo.Connect(connectCtx, opts)
	if err != nil {
		return nil, err
	}
	// listCollections needs authorization, unlike Connect and Ping, so wrong or missing
	// credentials fail here instead of at the first query.
	listOpts := options.ListCollections().SetNameOnly(true).SetAuthorizedCollections(true)
	if _, err = client.Database(conf.MongoDatabase).ListCollectionNames(connectCtx, bson.D{}, listOpts); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo startup check failed: %w", err)
	}
	db := &Mongo{config: conf, client: client}
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			client.Disconnect(context.Background())
			return nil, err
		}
	}
	go func() {
		<-ctx.Done()
		client.Disconnect(context.Background())
	}()
	return db, nil
}

func validateConfig(conf configuration.Config) error {
	if conf.MongoDatabase == "" {
		return errEmptyDatabase
	}
	if conf.MongoUser != "" && conf.MongoPassword == "" {
		return errMissingPassword
	}
	return nil
}

// clientOptions applies the credentials after the URI so they replace any given in MONGO_URL.
func clientOptions(conf configuration.Config) *options.ClientOptions {
	reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build() //ensure map marshalling to interface
	opts := options.Client().ApplyURI(conf.MongoUrl).SetRegistry(reg).SetMonitor(otelmongo.NewMonitor())
	if conf.MongoUser != "" {
		opts.SetAuth(options.Credential{
			Username:   conf.MongoUser,
			Password:   conf.MongoPassword,
			AuthSource: conf.MongoAuthSource,
		})
	}
	return opts
}

// getTimeoutContext derives a context with the mongo timeout from the given parent.
// context.WithoutCancel keeps the trace-context and the baggage of the parent, which is
// what the otelmongo monitor and the logging need, but not its cancellation: a caller that
// gives up must not cut off a write that is already running.
func getTimeoutContext(parent ...context.Context) (context.Context, context.CancelFunc) {
	if len(parent) > 0 && parent[0] != nil {
		return context.WithTimeout(context.WithoutCancel(parent[0]), 10*time.Second)
	}
	return context.WithTimeout(context.Background(), 10*time.Second)
}
