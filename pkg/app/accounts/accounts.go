// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package accounts

import (
	"net/http"

	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
)

type User interface {
	Username() string
}

type Manager interface {
	// Gets the user from the http request, typically from a cookie or another header.
	UserFromRequest(r *http.Request) (User, error)
	// Gives the account manager the chance to extract login information from the token (id token
	// for example), validate it, add cookies to the request, etc.
	OnOAuth2Exchange(w http.ResponseWriter, r *http.Request, idToken appOAuth2.IDTokenClaims) (User, error)
}

type AMType string

type Config struct {
	Type   AMType
	OAuth2 appOAuth2.OAuth2Config
}
