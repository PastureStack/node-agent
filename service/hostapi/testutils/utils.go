package testutils

import (
	"github.com/rancher/websocket-proxy/proxy"
	wspTestutils "github.com/rancher/websocket-proxy/testutils"
)

func ParseTestPrivateKey() interface{} {
	return wspTestutils.ParseTestPrivateKey()
}

func ParseTestPublicKey() interface{} {
	return wspTestutils.ParseTestPublicKey()
}

func GetTestConfig(addr string) *proxy.Config {
	config := &proxy.Config{
		ListenAddr: addr,
		CattleAddr: "127.0.0.1:8081",
	}

	config.PublicKey = ParseTestPublicKey()
	return config
}
