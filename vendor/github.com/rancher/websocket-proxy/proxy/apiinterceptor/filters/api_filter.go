package filters

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"

	"github.com/rancher/websocket-proxy/proxy/apiinterceptor/model"
	log "github.com/sirupsen/logrus"
)

type APIFilter interface {
	GetType() string
	ProcessFilter(filter model.FilterData, input model.APIRequestData) (model.APIRequestData, error)
}

func SignString(stringToSign []byte, sharedSecret []byte) string {
	h := hmac.New(sha512.New, sharedSecret)
	h.Write(stringToSign)

	signature := h.Sum(nil)
	encodedSignature := base64.URLEncoding.EncodeToString(signature)

	log.Debugf("Signature generated: %v", encodedSignature)

	return encodedSignature
}
