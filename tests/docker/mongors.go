/*
 * Copyright 2021 InfAI (CC SES)
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

package docker

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"sync"
	"time"
)

func MongoRs(ctx context.Context, wg *sync.WaitGroup) (mongourl string, err error) {
	log.Println("start mongo-rs")

	hostPort := "27017"
	hostIp, err := getHostIp()
	if err != nil {
		return "", err
	}

	mongourl = "mongodb://" + hostIp + ":" + hostPort

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Dockerfile:    "mongo-rs",
				Context:       "./docker/dockerfiles",
				Repo:          "mongors",
				PrintBuildLog: true,
			},
			ExposedPorts: []string{"27017:27017"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("27017/tcp"),
				wait.ForNop(waitretry(1*time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					log.Println("try mongodb connection...")
					ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
					client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongourl))
					if err != nil {
						log.Println("connection-error:", err)
						return err
					}
					err = client.Ping(ctx, readpref.Primary())
					if err != nil {
						log.Println("ping-error:", err)
						return err
					}
					return nil
				}))),
		},
		Started: true,
	})
	if err != nil {
		return "", err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container mongo-rs", c.Terminate(context.Background()))
	}()

	return mongourl, nil
}
