package util

import (
	"github.com/PastureStack/node-agent/service/hostapi/config"
	rclient "github.com/rancher/go-rancher/client"
)

func GetPlatformClient() (*rclient.RancherClient, error) {
	apiURL := config.Config.PlatformURL
	accessKey := config.Config.PlatformAccessKey
	secretKey := config.Config.PlatformSecretKey

	if apiURL == "" || accessKey == "" || secretKey == "" {
		return nil, nil
	}

	apiClient, err := rclient.NewRancherClient(&rclient.ClientOpts{
		Url:       apiURL,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return nil, err
	}
	return apiClient, nil
}
