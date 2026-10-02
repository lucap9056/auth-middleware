package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Body[T any] struct {
	Success bool `json:"success"`
	Message T    `json:"message"`
}

func JSON[T any](w http.ResponseWriter, success bool, message T, code int, internalErr error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if internalErr != nil {
		log.Printf("[ERROR] Status: %d, Msg: %v, Err: %v", code, message, internalErr)
	}

	json.NewEncoder(w).Encode(Body[T]{
		Success: success,
		Message: message,
	})
}

const (
	BearerChallenge       = `Bearer`
	InvalidTokenChallenge = `Bearer error="invalid_token"`
)

func Unauthorized(w http.ResponseWriter, challenge, message string, internalErr error) {
	w.Header().Set("WWW-Authenticate", challenge)
	JSON(w, false, message, http.StatusUnauthorized, internalErr)
}
