package auth

import (
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
)

const expectedJWTAlg = "RS256"

func ParseToken(tokenString string, parsedPublicKey interface{}) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("no JWT token provided")
	}

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		method, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok || method.Alg() != expectedJWTAlg {
			return nil, fmt.Errorf("unexpected JWT signing method: %v", token.Header["alg"])
		}
		return parsedPublicKey, nil
	})
}

func GetClaim(token *jwt.Token, key string) (interface{}, bool) {
	claims, ok := MapClaims(token)
	if !ok {
		return nil, false
	}
	value, found := claims[key]
	return value, found
}

func GetClaimString(token *jwt.Token, key string) (string, bool) {
	value, found := GetClaim(token, key)
	if !found {
		return "", false
	}
	valueString, ok := value.(string)
	return valueString, ok
}

func GetClaimMap(token *jwt.Token, key string) (map[string]interface{}, bool) {
	value, found := GetClaim(token, key)
	if !found {
		return nil, false
	}
	valueMap, ok := value.(map[string]interface{})
	return valueMap, ok
}

func MapClaims(token *jwt.Token) (jwt.MapClaims, bool) {
	if token == nil {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	return claims, ok
}
