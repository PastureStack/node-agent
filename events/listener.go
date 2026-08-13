package events

import (
	"os"
	"path/filepath"
	"time"

	"github.com/PastureStack/node-agent/handlers"
	"github.com/PastureStack/node-agent/service/hostapi"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/pkg/errors"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/log"
)

func Listen(eventURL, accessKey, secretKey string, workerCount int) error {
	log.Infof("Listening for events on %v", eventURL)

	config.SetAccessKey(accessKey)
	config.SetSecretKey(secretKey)
	config.SetAPIURL(eventURL)

	config.PhysicalHostUUID(true)
	config.SetDockerUUID()

	log.Info("launching hostapi")
	go hostapi.StartUp()

	go func() {
		timestamps := time.Time{}
		for {
			if !checkTS(&timestamps) {
				log.Info("timestamp files have been changed. Exiting go-agent")
				os.Exit(1)
			}
			time.Sleep(time.Duration(2) * time.Second)
		}
	}()

	eventHandlers, err := handlers.GetHandlers()
	if err != nil {
		return errors.Wrap(err, "Failed to get event handlers")
	}

	pingConfig := revents.PingConfig{
		SendPingInterval:  5000,
		CheckPongInterval: 5000,
		MaxPongWait:       60000,
	}
	router, err := revents.NewEventRouter("", 0, eventURL, accessKey, secretKey, nil, eventHandlers, "", workerCount, pingConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to create new event router")
	}
	err = router.StartWithoutCreate(nil)
	if err != nil {
		return errors.Wrap(err, "Error encountered while running event router")
	}
	return nil
}

func checkTS(timestamps *time.Time) bool {
	stampFile := config.Stamp()
	root, leaf := filepath.Split(stampFile)
	directory, openErr := os.OpenRoot(filepath.Clean(root))
	if openErr != nil {
		return true
	}
	defer directory.Close()
	stats, err := directory.Stat(leaf)
	if err != nil {
		return true
	}
	ts := stats.ModTime()
	// check whether timestamps has been initialized
	if timestamps.IsZero() {
		*timestamps = ts
	}
	return timestamps.Equal(ts)
}
