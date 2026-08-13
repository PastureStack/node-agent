//go:build windows
// +build windows

package register

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/log"
)

const (
	platformAgentIP   = "PLATFORM_AGENT_IP"
	platformURLEnv    = "PLATFORM_URL"
	tokenFile         = "C:/ProgramData/PastureStack/registrationToken"
	platformAccessKey = "PLATFORM_ACCESS_KEY"
	platformSecretKey = "PLATFORM_SECRET_KEY"
	apiCrtFile        = "C:/ProgramData/PastureStack/etc/platform/api.crt"
)

func RunRegistration(url string) error {
	accessKey, secretKey, platformURL, agentIP := loadEnv(url)
	os.Setenv(platformAgentIP, agentIP)
	os.Setenv(platformURLEnv, platformURL)
	// Historical names remain populated only for wire-compatible child processes.
	os.Setenv("CATTLE_AGENT_IP", agentIP)
	os.Setenv("CATTLE_URL", platformURL)
	if err := downloadAPICrt(); err != nil {
		return err
	}
	return register(accessKey, secretKey, platformURL)
}

func loadEnv(url string) (string, string, string, string) {
	accessKey, secretKey, platformURL, agentIP := "", "", "", ""
	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", ""
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "CATTLE_REGISTRATION_ACCESS_KEY") {
			str := strings.Split(line, "=")[1]
			accessKey = str[1 : len(str)-1]
		} else if strings.Contains(line, "CATTLE_REGISTRATION_SECRET_KEY") {
			str := strings.Split(line, "=")[1]
			secretKey = str[1 : len(str)-1]
		} else if strings.Contains(line, "CATTLE_URL") {
			str := strings.Split(line, "=")[1]
			platformURL = str[1 : len(str)-1]
		} else if strings.Contains(line, "DETECTED_CATTLE_AGENT_IP") {
			if envAgentIP := os.Getenv("CATTLE_AGENT_IP"); envAgentIP != "" {
				agentIP = envAgentIP
			} else {
				str := strings.Split(line, "=")[1]
				agentIP = str[1 : len(str)-1]
			}
		}
	}
	return accessKey, secretKey, platformURL, agentIP
}

func register(accessKey, secretKey, platformURL string) error {
	token, err := getToken()
	if err != nil {
		return err
	}
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Timeout:   time.Second * 30,
		Url:       platformURL,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return err
	}
	resp, err := apiClient.Register.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"key": token,
		},
	})
	if err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		_, err := apiClient.Register.Create(&client.Register{
			Key: token,
		})
		if err != nil {
			return err
		}
		i := 0
		for {
			if i == 10 {
				return errors.New("Failed to genarate access key")
			}
			list, err := apiClient.Register.List(&client.ListOpts{
				Filters: map[string]interface{}{
					"key": token,
				},
			})
			if err != nil {
				return err
			}
			if len(list.Data) == 0 || list.Data[0].AccessKey == "" {
				time.Sleep(time.Second)
				i++
				continue
			}
			os.Setenv(platformAccessKey, list.Data[0].AccessKey)
			os.Setenv(platformSecretKey, list.Data[0].SecretKey)
			os.Setenv("CATTLE_ACCESS_KEY", list.Data[0].AccessKey)
			os.Setenv("CATTLE_SECRET_KEY", list.Data[0].SecretKey)
			break
		}

	} else {
		list, err := apiClient.Register.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"key": token,
			},
		})
		if err != nil {
			return err
		}
		os.Setenv(platformAccessKey, list.Data[0].AccessKey)
		os.Setenv(platformSecretKey, list.Data[0].SecretKey)
		os.Setenv("CATTLE_ACCESS_KEY", list.Data[0].AccessKey)
		os.Setenv("CATTLE_SECRET_KEY", list.Data[0].SecretKey)
	}
	return nil
}

func getToken() (string, error) {
	if _, err := os.Stat(tokenFile); err == nil {
		file, _ := os.Open(tokenFile)
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	file, err := os.Create(tokenFile)
	if err != nil {
		return "", err
	}
	defer file.Close()
	b := make([]byte, 64)
	rand.Read(b)
	token := fmt.Sprintf("%x", b)
	_, err = file.WriteString(token)
	if err != nil {
		return "", err
	}
	return token, nil
}

func downloadAPICrt() error {
	if _, err := os.Stat(apiCrtFile); err == nil {
		os.Remove(apiCrtFile)
	}
	if err := os.MkdirAll("C:/ProgramData/PastureStack/etc/platform", 0755); err != nil {
		return err
	}
	file, err := os.Create(apiCrtFile)
	if err != nil {
		return err
	}
	defer file.Close()
	response, err1 := http.Get(os.Getenv(platformURLEnv) + "/scripts/api.crt")
	if err1 != nil {
		log.Error(fmt.Sprintf("Error while downloading error: %s", err1))
		return err1
	}
	defer response.Body.Close()
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Error(fmt.Sprintf("Error while copy file: %s", err))
		return err
	}
	return nil
}
