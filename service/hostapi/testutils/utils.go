package testutils

import (
	"github.com/PastureStack/websocket-proxy/proxy"
	wspTestutils "github.com/PastureStack/websocket-proxy/testutils"
)

func ParseTestPrivateKey() interface{} {
	return wspTestutils.ParseTestPrivateKey()
}

func ParseTestPublicKey() interface{} {
	return wspTestutils.ParseTestPublicKey()
}

func GetTestConfig(addr string) *proxy.Config {
	config := &proxy.Config{
		ListenAddr:   addr,
		PlatformAddr: "127.0.0.1:8081",
	}

	config.PublicKey = ParseTestPublicKey()
	return config
}
