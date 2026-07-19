package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PastureStack/node-agent/cloudprovider"
	"github.com/PastureStack/node-agent/events"
	"github.com/PastureStack/node-agent/register"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/rancher/log"
	logserver "github.com/rancher/log/server"

	_ "github.com/PastureStack/node-agent/cloudprovider/aliyun"
	_ "github.com/PastureStack/node-agent/cloudprovider/aws"
)

var (
	VERSION = "dev"
)

func main() {
	logserver.StartServerWithDefaults()
	version := flag.Bool("version", false, "node-agent version")
	rurl := flag.String("url", "", "registration URL")
	registerService := flag.String("register-service", "", "register node-agent service")
	unregisterService := flag.Bool("unregister-service", false, "unregister node-agent service")
	locale := flag.String("locale", localeFromEnvironment(), "operator message locale: en-US or zh-TW")
	flag.Parse()
	if *locale != "en-US" && *locale != "zh-TW" {
		log.Fatalf("unsupported locale %q; use en-US or zh-TW", *locale)
	}
	if *version {
		fmt.Printf("node-agent version %s\n", VERSION)
		os.Exit(0)
	}

	if os.Getenv("PASTURESTACK_DEBUG") != "" || os.Getenv("CATTLE_SCRIPT_DEBUG") != "" || os.Getenv("RANCHER_DEBUG") != "" {
		log.SetLevelString("debug")
	}

	if err := register.Init(*registerService, *unregisterService); err != nil {
		log.Fatalf("Failed to Initialize Service err: %v", err)
	}

	if *rurl != "" {
		err := register.RunRegistration(*rurl)
		if err != nil {
			log.Errorf("registration failed. err: %v", err)
			os.Exit(1)
		}
	}

	log.Info(operatorMessage(*locale, "launch"))

	url := config.DefaultValue("URL", "")
	accessKey := config.DefaultValue("ACCESS_KEY", "")
	secretKey := config.DefaultValue("SECRET_KEY", "")
	workerCount := 250

	if config.DetectCloudProvider() {
		log.Info("Detecting cloud provider")
		cloudprovider.GetCloudProviderInfo()
	}

	err := events.Listen(url, accessKey, secretKey, workerCount)
	if err != nil {
		log.Fatalf("Exiting. Error: %v", err)
		register.NotifyShutdown(err)
	}
}

func localeFromEnvironment() string {
	if locale := os.Getenv("PASTURESTACK_LOCALE"); locale != "" {
		return locale
	}
	return "en-US"
}

func operatorMessage(locale, key string) string {
	messages := map[string]map[string]string{
		"en-US": {"launch": "Launching PastureStack node agent"},
		"zh-TW": {"launch": "正在啟動 PastureStack 節點代理程式"},
	}
	return messages[locale][key]
}
