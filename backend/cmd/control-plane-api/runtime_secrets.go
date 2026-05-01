package main

import (
	"github.com/dnviti/arsenale/backend/internal/desktopbroker"
	"github.com/dnviti/arsenale/backend/internal/modelgateway"
)

type runtimeSecrets struct {
	GuacamoleSecret     string
	JWTSecret           string
	GuacencAuthToken    string
	ServerEncryptionKey []byte
}

func loadRuntimeSecrets() (runtimeSecrets, error) {
	guacamoleSecret, err := desktopbroker.LoadSecret("GUACAMOLE_SECRET", "GUACAMOLE_SECRET_FILE")
	if err != nil {
		return runtimeSecrets{}, err
	}
	jwtSecret, err := desktopbroker.LoadSecret("JWT_SECRET", "JWT_SECRET_FILE")
	if err != nil {
		return runtimeSecrets{}, err
	}
	guacencAuthToken, err := loadOptionalSecret("GUACENC_AUTH_TOKEN", "GUACENC_AUTH_TOKEN_FILE")
	if err != nil {
		return runtimeSecrets{}, err
	}
	serverEncryptionKey, err := modelgateway.LoadServerEncryptionKey()
	if err != nil {
		return runtimeSecrets{}, err
	}
	return runtimeSecrets{
		GuacamoleSecret:     guacamoleSecret,
		JWTSecret:           jwtSecret,
		GuacencAuthToken:    guacencAuthToken,
		ServerEncryptionKey: serverEncryptionKey,
	}, nil
}
