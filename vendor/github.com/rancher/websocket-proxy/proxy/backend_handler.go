package proxy

import (
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type BackendHandler struct {
	proxyManager    proxyManager
	parsedPublicKey interface{}
}

func (h *BackendHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Infof("Handling backend connection request.")
	hostKey, authed := h.auth(req)
	if !authed {
		http.Error(rw, "Failed authentication", 401)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Errorf("Error during upgrade for host [%v]: [%v]", hostKey, err)
		http.Error(rw, "Failed to upgrade connection.", 500)
		return
	}

	h.proxyManager.addBackend(hostKey, ws)
}

func (h *BackendHandler) auth(req *http.Request) (string, bool) {
	token, tokenParam, err := parseToken(req, h.parsedPublicKey)
	if err != nil {
		log.Warnf("Error parsing backend token: %v. Failing auth. Token parameter: %v", err, redactSecretForLog(tokenParam))
		return "", false
	}

	reportedUUID, found := claimValue(token, "reportedUuid")
	if !found {
		log.Warnf("Token did not have a reportedUuid. Failing auth. Token parameter: %v", redactSecretForLog(tokenParam))
		return "", false
	}

	hostKey, ok := reportedUUID.(string)
	if !ok || hostKey == "" {
		log.Warnf("Token's reported uuid claim %v could not be parsed as a string. Token parameter: %v", reportedUUID, redactSecretForLog(tokenParam))
		return "", false
	}

	return hostKey, true
}
