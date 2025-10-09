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
