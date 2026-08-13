package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
)

const expectedJWTAlg = "RS256"

func parseSignedJWT(tokenString string, parsedPublicKey interface{}) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		method, ok := token.Method.(*jwt.SigningMethodRSA)
		if !ok || method.Alg() != expectedJWTAlg {
			return nil, fmt.Errorf("unexpected JWT signing method: %v", token.Header["alg"])
		}
		return parsedPublicKey, nil
	})
}

func mapClaims(token *jwt.Token) (jwt.MapClaims, bool) {
	if token == nil {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	return claims, ok
}

func stringClaim(token *jwt.Token, key string) (string, bool) {
	value, found := claimValue(token, key)
	if !found {
		return "", false
	}
	valueString, ok := value.(string)
	return valueString, ok
}

func boolClaim(token *jwt.Token, key string) (bool, bool) {
	value, found := claimValue(token, key)
	if !found {
		return false, false
	}
	valueBool, ok := value.(bool)
	return valueBool, ok
}

func objectClaim(token *jwt.Token, key string) (map[string]interface{}, bool) {
	value, found := claimValue(token, key)
	if !found {
		return nil, false
	}
	valueMap, ok := value.(map[string]interface{})
	return valueMap, ok
}

func claimValue(token *jwt.Token, key string) (interface{}, bool) {
	claims, ok := mapClaims(token)
	if !ok {
		return nil, false
	}
	value, found := claims[key]
	if !found {
		return nil, false
	}
	return value, true
}

func redactSecretForLog(value string) string {
	if value == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("[redacted len=%d sha256=%s]", len(value), hex.EncodeToString(sum[:])[:12])
}
