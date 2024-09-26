package main

import (
	"fmt"
	"log"
	"monitor_watch/lib"
	"strings"
	"time"
)

const version = "v1.4"

func main() {
	log.Println("monitor_watch", version, "started")

	// TODO Add fsnotify
	for {
		handleDisplayChange()
		time.Sleep(10 * time.Second)
	}
}

func handleDisplayChange() {
	// Always get main display ids, because they can change for example with Remote Desktop
	mainDisplayIsActive := false
	fakeDisplayIsActive := false
	sunshineIsStreaming := false

	response := lib.GetMultiMonitorDeviceResponse()
	mainDisplayIds := response.GetMainDisplayIds()

	mainDisplayIsActive = response.IsMainDisplayActive(mainDisplayIds)
	fakeDisplayIsActive = response.IsFakeDisplayActive()
	sunshineIsStreaming = lib.IsSunshineStreaming()

	logline := fmt.Sprintf("Fake display %s is %s. ", lib.FakeDisplayId, formatDeviceState(fakeDisplayIsActive))
	logline += fmt.Sprintf("Main displays %s are %s. ", strings.Join(mainDisplayIds, ","), formatDeviceState(mainDisplayIsActive))
	logline += fmt.Sprintf("Sunshine is %s\n", formatSunshineState(sunshineIsStreaming))
	log.Printf(logline)

	if sunshineIsStreaming && (!fakeDisplayIsActive || mainDisplayIsActive) {
		log.Println("Sunshine is streaming and main display is active. Deactivate Main Display, activate fake display.")
		lib.DisableDisplay(mainDisplayIds)
		lib.EnableDisplay([]string{lib.FakeDisplayId})
		// since the fake display is activated now, we can safely switch the resolution to that of the client
		resolution, framerate := lib.GetDesiredResolutionAndFramerate()
		log.Printf("Changing desired Resolution: %s, Framerate: %s\n", resolution, framerate)
		err := lib.ChangeResolutionAndFramerate(resolution, framerate)
		if err != nil {
			log.Fatal(err)
		}
	}

	if !sunshineIsStreaming && (fakeDisplayIsActive || !mainDisplayIsActive) {
		log.Println("Sunshine is not streaming and fake display is active. Activate Main Display, deactivate Fake display")
		lib.DisableDisplay([]string{lib.FakeDisplayId})
		lib.EnableDisplay(mainDisplayIds)
	}
}

func formatSunshineState(streaming bool) string {
	if streaming {
		return "streaming"
	}
	return "not streaming"
}

func formatDeviceState(active bool) string {
	if active {
		return "active"
	}
	return "inactive"
}
