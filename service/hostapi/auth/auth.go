package auth

import (
	"net/http"

	"github.com/PastureStack/node-agent/service/hostapi/app/common"
	"github.com/PastureStack/node-agent/service/hostapi/config"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/golang/glog"
)

func Auth(rw http.ResponseWriter, req *http.Request) bool {
	if !config.Config.Auth {
		return true
	}
	tokenString := req.URL.Query().Get("token")

	if len(tokenString) == 0 {
		return false
	}

	token, err := ParseToken(tokenString, config.Config.ParsedPublicKey)
	SetToken(req, token)

	if err != nil {
		common.CheckError(err, 2)
		return false
	}

	if !token.Valid {
		return false
	}

	if config.Config.HostUUIDCheck {
		hostUUID, found := GetClaimString(token, "hostUuid")
		if !found || hostUUID != config.Config.HostUUID {
			glog.Infoln("Host UUID mismatch , authentication failed")
			return false
		}
	}

	return true
}

func GetAndCheckToken(tokenString string) (*jwt.Token, bool) {
	token, err := ParseToken(tokenString, config.Config.ParsedPublicKey)
	if err != nil {
		common.CheckError(err, 2)
		return token, false
	}

	if !token.Valid {
		return token, false
	}

	if config.Config.HostUUIDCheck {
		hostUUID, found := GetClaimString(token, "hostUuid")
		if !found || hostUUID != config.Config.HostUUID {
			glog.Infoln("Host UUID mismatch , authentication failed")
			return token, false
		}
	}

	return token, true

}
