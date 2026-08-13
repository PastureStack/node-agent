package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"

	jwt "github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
)

var (
	testKeyOnce    sync.Once
	testPrivateKey *rsa.PrivateKey
)

func getTestPrivateKey() *rsa.PrivateKey {
	testKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatal("Failed to generate test private key.", err)
		}
		testPrivateKey = key
	})
	return testPrivateKey
}

func CreateTokenWithPayload(payload map[string]interface{}, privateKey interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(payload))
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

func CreateToken(hostUUID string, privateKey interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"hostUuid": hostUUID,
	})
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

func CreateBackendToken(reportedUUID string, privateKey interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"reportedUuid": reportedUUID,
	})
	signed, err := token.SignedString(privateKey)
	if err != nil {
		log.Fatal("Failed to parse private key.", err)
	}
	return signed
}

func ParseTestPrivateKey() interface{} {
	return getTestPrivateKey()
}

func ParseTestPublicKey() interface{} {
	return &getTestPrivateKey().PublicKey
}
