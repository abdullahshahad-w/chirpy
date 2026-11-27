package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(header http.Header) (string, error) {
		authHeader := header.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("Authorization header doesn't exist")
	}

	fields := strings.Fields(authHeader)

	if len(fields) != 2 || fields[0] != "ApiKey" {
		return "", errors.New("Bearer not here mate")
	}

	return fields[1], nil
}