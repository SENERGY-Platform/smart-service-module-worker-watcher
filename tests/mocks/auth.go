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

package mocks

import (
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/auth"
	"github.com/golang-jwt/jwt"
	"runtime/debug"
	"strings"
	"time"
)

type AuthMock struct{}

func (this AuthMock) ExchangeUserToken(userid string) (token auth.Token, err error) {
	token = auth.Token{
		Sub:         userid,
		RealmAccess: nil,
	}
	token.Token, err = this.GenerateUserTokenById(userid)
	return token, err
}

type KeycloakClaims struct {
	RealmAccess RealmAccess `json:"realm_access"`
	jwt.StandardClaims
}

type RealmAccess struct {
	Roles []string `json:"roles"`
}

func (this AuthMock) GenerateUserTokenById(userid string) (token string, err error) {
	roles := []string{"user"}
	claims := KeycloakClaims{
		RealmAccess{Roles: roles},
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
			Issuer:    "test",
			Subject:   userid,
		},
	}
	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	unsignedTokenString, err := jwtoken.SigningString()
	if err != nil {
		debug.PrintStack()
		return token, err
	}
	tokenString := strings.Join([]string{unsignedTokenString, ""}, ".")
	token = "Bearer " + tokenString
	return token, err
}
