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
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"sync"
	"time"
)

func MongoRs(ctx context.Context, wg *sync.WaitGroup) (mongourl string, err error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", err
	}

	networks, _ := pool.Client.ListNetworks()
	hostIp := ""
	for _, network := range networks {
		if network.Name == "bridge" {
			hostIp = network.IPAM.Config[0].Gateway
		}
	}

	mongourl = "mongodb://" + hostIp + ":27017"

	container, err := pool.BuildAndRunWithOptions("./docker/dockerfiles/mongo-rs", &dockertest.RunOptions{
		Name:         "mongors",
		ExposedPorts: []string{"27017/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{"27017/tcp": {{HostPort: "27017"}}},
	}, func(config *docker.HostConfig) {
		config.Tmpfs = map[string]string{"/data/db": "rw"}
	})

	if err != nil {
		return "", err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()

	go Dockerlog(pool, ctx, container, "MONGO")

	err = pool.Retry(func() error {
		log.Println("try mongodb connection...")
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
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
	})
	return mongourl, err
}
