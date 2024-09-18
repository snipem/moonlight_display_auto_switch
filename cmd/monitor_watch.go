package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const fakeDisplayId = "LNX0000"
const version = "v1.3"

func main() {
	log.Println("monitor_watch", version, "started")
	mainDisplayIsActive := false
	fakeDisplayIsActive := false
	sunshineIsStreaming := false

	for {
		// Always get main display ids, because they can change for example with Remote Desktop
		mainDisplayIds := getMainDisplayIds()

		mainDisplayIsActive = isMainDisplayActive(mainDisplayIds)
		fakeDisplayIsActive = isFakeDisplayActive()
		sunshineIsStreaming = isSunshineStreaming()

		log.Printf("Fake display %s is active: %v\n", fakeDisplayId, fakeDisplayIsActive)
		log.Printf("Main displays %s are active: %v\n", strings.Join(mainDisplayIds, ","), mainDisplayIsActive)
		log.Println("Sunshine is streaming according to log: ", sunshineIsStreaming)

		if sunshineIsStreaming && (!fakeDisplayIsActive || mainDisplayIsActive) {
			log.Println("Sunshine is streaming and main display is active. Deactivate Main Display, activate fake display.")
			disableDisplay(mainDisplayIds)
			enableDisplay([]string{fakeDisplayId})
			// since the fake display is activated now, we can safely switch the resolution to that of the client
			resolution, framerate := getDesiredResolutionAndFramerate()
			log.Printf("EXPERIMENTAL: Desired Resolution: %s, Framerate: %s\n", resolution, framerate)
			//err := changeResolutionAndFramerate(resolution, framerate)
			//if err != nil {
			//	log.Fatal(err)
			//}
		}

		if !sunshineIsStreaming && (fakeDisplayIsActive || !mainDisplayIsActive) {
			log.Println("Sunshine is not streaming and fake display is active. Activate Main Display, deactivate Fake display")
			disableDisplay([]string{fakeDisplayId})
			enableDisplay(mainDisplayIds)
		}

		time.Sleep(10 * time.Second)
	}
}

func changeResolutionAndFramerate(resolution string, framerate string) error {
	// C:\Users\mail\bin\multimonitortool-x64\MultiMonitorTool.exe /SetMonitors "Name=LNX0000 Width=%SUNSHINE_CLIENT_WIDTH% Height=%SUNSHINE_CLIENT_HEIGHT% DisplayFrequency=%SUNSHINE_CLIENT_FPS%"

	width, height := resolutionToWidthAndHeight(resolution)

	cmd := exec.Command("MultiMonitorTool.exe", "/SetMonitors", fmt.Sprintf("Name=%s Width=%s Height=%s DisplayFrequency=%s", fakeDisplayId, width, height, framerate))
	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}
	return nil
}

func resolutionToWidthAndHeight(resolution string) (string, string) {
	val := strings.Split(resolution, "x")
	return val[0], val[1]
}

func getMainDisplayIds() []string {
	mainDisplayIds := []string{}
	response := getMultiMonitorDeviceResponse()
	for i, line := range response {
		if i != 0 {
			displayId := line[multiMonitorResponseDisplayNameField]
			if displayId != fakeDisplayId {
				mainDisplayIds = append(mainDisplayIds, displayId)
			}
		}
	}
	return mainDisplayIds
}

func readLogFile(path string) (string, error) {
	// Expand environment variables in the path
	expandedPath := os.ExpandEnv(path)

	// Open the log file
	file, err := os.Open(expandedPath)
	if err != nil {
		return "", fmt.Errorf("could not open log file: %w", err)
	}
	defer file.Close()

	// Read the file content
	var content string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading log file: %w", err)
	}

	return content, nil
}

var sunshineLogFile = "${ProgramFiles}\\Sunshine\\config\\sunshine.log"

func isSunshineStreaming() bool {
	logfile, err := readLogFile(sunshineLogFile)
	if err != nil {
		log.Fatal(err)
	}
	isConnected := false

	for _, logline := range strings.Split(logfile, "\n") {

		if strings.Contains(logline, "CLIENT CONNECTED") {
			isConnected = true
		}

		if strings.Contains(logline, "CLIENT DISCONNECTED") {
			isConnected = false
		}

	}

	return isConnected
}

func getDesiredResolutionAndFramerate() (string, string) {
	logfile, err := readLogFile(sunshineLogFile)
	if err != nil {
		log.Fatal(err)
	}
	resolution := ""
	framerate := ""
	for _, logline := range strings.Split(logfile, "\n") {
		// TODO maybe check that these lines are successive
		if strings.Contains(logline, "Desktop resolution [") {
			r, err := regexp.Compile(`[0-9]+x[0-9]+`)
			if err != nil {
				log.Fatal(err)
			}
			resolution = r.FindString(logline)
		}
		if strings.Contains(logline, "Requested frame rate [") {
			r, err := regexp.Compile(`([0-9]+)fps`)
			if err != nil {
				log.Fatal(err)
			}
			// get first group
			framerate = r.FindStringSubmatch(logline)[1]
		}
	}
	return resolution, framerate
}

func isFakeDisplayActive() bool {
	displayName := fakeDisplayId
	return isDisplayActive(displayName)
}

func isMainDisplayActive(mainDisplayIds []string) bool {
	returnValue := false
	for _, id := range mainDisplayIds {
		if !isDisplayActive(id) {
			// immediately return false if one display is not active
			return false
		}
		returnValue = true

	}
	return returnValue
}

func disableDisplay(monitor []string) {
	err := changeDisplay("/disable", monitor)
	if err != nil {
		log.Fatal(err)
	}
}

func enableDisplay(monitor []string) {
	err := changeDisplay("/enable", monitor)
	if err != nil {
		log.Fatal(err)
	}
}

// changeDisplay executes MultiMonitorTool with the given command and
// display ID(s), and returns an error if any of the commands fail.
//
// The command should be one of "/enable" or "/disable".
func changeDisplay(command string, displays []string) error {

	for _, display := range displays {
		// Create the command with arguments
		cmd := exec.Command("MultiMonitorTool.exe", command, display)

		// Run the command
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute command: %v", err)
		}
	}
	return nil
}

// value 3 is active state
// value 15 is name
const multiMonitorResponseActiveStateField = 3
const multiMonitorResponseDisplayNameField = 15

// isDisplayActive checks if a display is currently active.
//
// displayName is the name of the display as it is known by the MultiMonitorTool.
// The function returns true if the display is active and false otherwise.
func isDisplayActive(displayName string) bool {
	response := getMultiMonitorDeviceResponse()
	for i, record := range response {
		if i > 0 {
			if record[multiMonitorResponseActiveStateField] == "Yes" && record[multiMonitorResponseDisplayNameField] == displayName {
				return true
			}
		}
	}
	return false
}

func getMultiMonitorDeviceResponse() [][]string {
	// Generate a random file name for outputting the multimonitor device response
	tempFolder := os.TempDir()
	tempFile := filepath.Join(tempFolder, fmt.Sprintf("multimonitortool_%d.csv", rand.Int()))

	response, err := runCommandAndParseCSV("MultiMonitorTool.exe", "/scomma", tempFile)
	if err != nil {
		log.Fatalf("Error running multimonitortool: %s", err)
	}

	// Remove the temporary file
	err = os.Remove(tempFile)
	if err != nil {
		log.Fatalf("Error removing temporary file: %s", err)
	}

	return response
}

func runCommandAndParseCSV(executable, args, filename string) ([][]string, error) {
	// Create the command with arguments
	cmd := exec.Command(executable, args, filename)

	// Run the command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", err)
	}

	// Open the CSV file
	csvFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

	// Create a new CSV reader
	reader := csv.NewReader(csvFile)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %v", err)
	}

	return records, nil
}
