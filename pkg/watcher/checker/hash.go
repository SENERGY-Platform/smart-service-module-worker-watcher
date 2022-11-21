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

package checker

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

const HASH_TYPE_MD5 = "md5"
const HASH_TYPE_SHA256 = "sha256"
const HASH_TYPE_DEVICEIDS = "deviceids"

func hash(hashType string, payload []byte) (string, error) {
	switch hashType {
	case HASH_TYPE_MD5:
		return fmt.Sprintf("%x", md5.Sum(payload)), nil
	case HASH_TYPE_SHA256:
		return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
	case HASH_TYPE_DEVICEIDS:
		return deviceIdsHash(payload)
	default:
		return fmt.Sprintf("%x", md5.Sum(payload)), nil
	}
}

func deviceIdsHash(payload []byte) (string, error) {
	deviceIds, err := findDeviceIds(payload)
	if err != nil {
		return "", err
	}
	temp, err := json.Marshal(deviceIds)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(temp)), nil
}

func findDeviceIds(payload []byte) (ids []string, err error) {
	exp := "urn:infai:ses:device:[0-9a-x-]{36}"
	ids = regexp.MustCompile(exp).FindAllString(string(payload), -1)
	sort.Strings(ids)
	return ids, nil
}
