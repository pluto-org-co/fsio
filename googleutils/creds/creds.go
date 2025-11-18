// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package creds

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

//go:embed account.txt
var userEmail string

func UserEmail() (s string) {
	return strings.TrimSpace(userEmail)
}

//go:embed svc-account.json
var SvcAccount []byte

func NewConfiguration(t *testing.T, scopes ...string) (conf *jwt.Config) {
	assertions := assert.New(t)

	conf, err := google.JWTConfigFromJSON(SvcAccount, scopes...)
	if !assertions.Nil(err, "failed to load json") {
		return nil
	}
	return conf
}
