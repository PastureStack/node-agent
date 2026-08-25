package auth

import (
	"context"
	"net/http"

	jwt "github.com/golang-jwt/jwt/v5"
)

type key int

const TokenKey key = 0

func GetToken(r *http.Request) *jwt.Token {
	if rv := r.Context().Value(TokenKey); rv != nil {
		return rv.(*jwt.Token)
	}
	return nil
}

func SetToken(r *http.Request, val *jwt.Token) {
	*r = *r.WithContext(context.WithValue(r.Context(), TokenKey, val))
}
