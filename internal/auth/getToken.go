package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("Authorization header doesn't exist")
	}

	fields := strings.Fields(authHeader)

	if len(fields) != 2 || fields[0] != "Bearer" {
		return "", errors.New("Bearer not here mate")
	}

	return fields[1], nil
}