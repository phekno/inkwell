// Package config exposes the inkwell deployment IDs to the TUI. Values are
// non-secret resource identifiers; env vars can override for local dev.
package config

import "os"

type Config struct {
	APIURL     string
	Region     string
	UserPoolID string
	ClientID   string
}

func Load() Config {
	return Config{
		APIURL:     env("INKWELL_API_URL", "https://svrge7paf2.execute-api.us-east-1.amazonaws.com"),
		Region:     env("INKWELL_REGION", "us-east-1"),
		UserPoolID: env("INKWELL_USER_POOL_ID", "us-east-1_9d6ekf6NA"),
		ClientID:   env("INKWELL_CLIENT_ID", "40mit3pvda2mi1640hrmij98bl"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
