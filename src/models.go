package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type siteConfig struct {
	EncryptionSecret string
	DatabasePath     string
	ListenURL        string
	VerboseErrors    bool
	IsDevelopment    bool
	CookieAge        int
}

func (config siteConfig) CookieAgeDuration() time.Duration {
	return time.Minute * time.Duration(config.CookieAge)
}

type user struct {
	DisplayName  string `json:"name"`
	EmailAddress string `json:"email"`
}

func (currentUser user) setUserCookie(w http.ResponseWriter) {
	currentUserJSON, _ := json.Marshal(currentUser)
	setEncryptedCookie("user", string(currentUserJSON), w)
}
