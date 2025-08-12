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

package docker

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func GetFreePortString() (string, error) {
	port, err := GetFreePort()
	if err != nil {
		return "", err
	}
	return strconv.Itoa(port), nil
}

func waitretry(timeout time.Duration, f func(ctx context.Context, target wait.StrategyTarget) error) func(ctx context.Context, target wait.StrategyTarget) error {
	return func(ctx context.Context, target wait.StrategyTarget) (err error) {
		return retry(timeout, func() error {
			return f(ctx, target)
		})
	}
}

func retry(timeout time.Duration, f func() error) (err error) {
	err = errors.New("initial")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
		if err != nil {
			wait := time.Duration(i) * time.Second
			if time.Since(start)+wait < timeout {
				time.Sleep(wait)
			} else {
				time.Sleep(time.Since(start) + wait - timeout)
				return f()
			}
		}
	}
	return err
}

func getHostIp() (string, error) {
	provider, err := testcontainers.NewDockerProvider(testcontainers.DefaultNetwork("bridge"))
	if err != nil {
		return "", err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	return provider.GetGatewayIP(ctx)
}
