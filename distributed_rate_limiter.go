package main

import (
	"encoding/json"
	"time"

	"github.com/TykTechnologies/drl"
	"github.com/TykTechnologies/logrus"
)

var DRLManager = &drl.DRL{}

func SetupDRL() {
	drlManager := &drl.DRL{}
	drlManager.Init()
	drlManager.ThisServerID = NodeID + "|" + HostDetails.Hostname
	log.Debug("DRL: Setting node ID: ", drlManager.ThisServerID)
	DRLManager = drlManager
}

func StartRateLimitNotifications() {
	notificationFreq := config.DRLNotificationFrequency
	if notificationFreq == 0 {
		notificationFreq = 2
	}

	go func() {
		log.Info("Starting gateway rate imiter notifications...")
		for {
			if NodeID != "" {
				NotifyCurrentServerStatus()
			} else {
				log.Warning("Node not registered yet, skipping DRL Notification")
			}

			time.Sleep(time.Duration(notificationFreq) * time.Second)
		}
	}()
}

func getTagHash() string {
	th := ""
	for _, tag := range config.DBAppConfOptions.Tags {
		th += tag
	}
	return th
}

func NotifyCurrentServerStatus() {
	if !DRLManager.Ready {
		return
	}

	rate := GlobalRate.Rate()
	if rate == 0 {
		rate = 1
	}

	server := drl.Server{
		HostName:   HostDetails.Hostname,
		ID:         NodeID,
		LoadPerSec: rate,
		TagHash:    getTagHash(),
	}

	asJson, err := json.Marshal(server)
	if err != nil {
		log.Error("Failed to encode payload: ", err)
		return
	}

	n := Notification{
		Command: NoticeGatewayDRLNotification,
		Payload: string(asJson),
	}

	MainNotifier.Notify(n)
}

func OnServerStatusReceivedHandler(payload string) {
	serverData := drl.Server{}
	err := json.Unmarshal([]byte(payload), &serverData)
	if err != nil {
		log.WithFields(logrus.Fields{
			"prefix": "pub-sub",
		}).Error("Failed unmarshal server data: ", err)
		return
	}

	log.Debug("Received DRL data: ", serverData)

	if DRLManager.Ready {
		DRLManager.AddOrUpdateServer(serverData)
		log.Debug(DRLManager.Report())
	} else {
		log.Warning("DRL not ready, skipping this notification")
	}
}
