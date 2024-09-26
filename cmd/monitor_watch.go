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
	"strconv"
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

		response := getMultiMonitorDeviceResponse()
		mainDisplayIds := response.getMainDisplayIds()

		mainDisplayIsActive = response.IsMainDisplayActive(mainDisplayIds)
		fakeDisplayIsActive = response.IsFakeDisplayActive()
		sunshineIsStreaming = isSunshineStreaming()

		logline := fmt.Sprintf("Fake display %s is %s. ", fakeDisplayId, formatDeviceState(fakeDisplayIsActive))
		logline += fmt.Sprintf("Main displays %s are %s. ", strings.Join(mainDisplayIds, ","), formatDeviceState(mainDisplayIsActive))
		logline += fmt.Sprintf("Sunshine is %s\n", formatSunshineState(sunshineIsStreaming))
		log.Printf(logline)

		if sunshineIsStreaming && (!fakeDisplayIsActive || mainDisplayIsActive) {
			log.Println("Sunshine is streaming and main display is active. Deactivate Main Display, activate fake display.")
			disableDisplay(mainDisplayIds)
			enableDisplay([]string{fakeDisplayId})
			// since the fake display is activated now, we can safely switch the resolution to that of the client
			resolution, framerate := getDesiredResolutionAndFramerate()
			log.Printf("Changing desired Resolution: %s, Framerate: %s\n", resolution, framerate)
			err := changeResolutionAndFramerate(resolution, framerate)
			if err != nil {
				log.Fatal(err)
			}
		}

		if !sunshineIsStreaming && (fakeDisplayIsActive || !mainDisplayIsActive) {
			log.Println("Sunshine is not streaming and fake display is active. Activate Main Display, deactivate Fake display")
			disableDisplay([]string{fakeDisplayId})
			enableDisplay(mainDisplayIds)
		}

		time.Sleep(10 * time.Second)
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

func changeResolutionAndFramerate(resolution string, framerate string) error {
	// C:\Users\mail\bin\multimonitortool-x64\MultiMonitorTool.exe /SetMonitors "Name=LNX0000 Width=%SUNSHINE_CLIENT_WIDTH% Height=%SUNSHINE_CLIENT_HEIGHT% DisplayFrequency=%SUNSHINE_CLIENT_FPS%"

	width, height := resolutionToWidthAndHeight(resolution)

	cmd := exec.Command("MultiMonitorTool.exe", "/SetMonitors", fmt.Sprintf("Name=%s Width=%d Height=%d DisplayFrequency=%s", fakeDisplayId, width, height, framerate))
	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}
	return nil
}

func resolutionToWidthAndHeight(resolution string) (int, int) {
	// Define the regular expression to match numbers
	re := regexp.MustCompile(`(\d+)\D+(\d+)`)

	// Find the first match in the string
	matches := re.FindStringSubmatch(resolution)

	// Convert the captured values from string to int
	if len(matches) == 3 {
		x, _ := strconv.Atoi(matches[1])
		y, _ := strconv.Atoi(matches[2])
		return x, y
	}

	// Return zeros if the pattern is not matched
	return -1, -1
}

func (response MultiMonitorDeviceResponse) getMainDisplayIds() []string {
	mainDisplayIds := []string{}
	for _, m := range response.MonitorInfo {
		if m.ShortMonitorID != fakeDisplayId {
			mainDisplayIds = append(mainDisplayIds, m.ShortMonitorID)
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

func (response MultiMonitorDeviceResponse) IsFakeDisplayActive() bool {
	return response.isDisplayActive(fakeDisplayId)
}

func (response MultiMonitorDeviceResponse) IsMainDisplayActive(mainDisplayIds []string) bool {
	returnValue := false
	for _, id := range mainDisplayIds {
		if !response.isDisplayActive(id) {
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

type MultiMonitorDeviceResponse struct {
	MonitorInfo []MonitorInfo
}

// isDisplayActive checks if a display is currently active.
//
// displayName is the name of the display as it is known by the MultiMonitorTool.
// The function returns true if the display is active and false otherwise.
func (response MultiMonitorDeviceResponse) isDisplayActive(displayName string) bool {
	for i, m := range response.MonitorInfo {
		if i > 0 {
			if m.Active == "Yes" && m.ShortMonitorID == displayName {
				return true
			}
		}
	}
	return false
}

type MonitorInfo struct {
	Resolution          string
	LeftTop             string
	RightBottom         string
	Active              string
	Disconnected        string
	Primary             string
	Colors              string
	Frequency           string
	Orientation         string
	MaximumResolution   string
	Name                string
	Adapter             string
	DeviceID            string
	DeviceKey           string
	MonitorID           string
	ShortMonitorID      string
	MonitorKey          string
	MonitorString       string
	MonitorName         string
	MonitorSerialNumber string
}

func (m MonitorInfo) IsActive() bool {
	return m.Active == "Yes"
}

func (m MonitorInfo) IsDisconnected() bool {
	return m.Disconnected == "Yes"
}

func (m MonitorInfo) IsPrimary() bool {
	return m.Primary == "Yes"
}

func (m MonitorInfo) GetCurrentResolution() (x int, y int) {
	return resolutionToWidthAndHeight(m.Resolution)
}

func getMultiMonitorDeviceResponse() MultiMonitorDeviceResponse {
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

	monitorInfos := []MonitorInfo{}
	for _, line := range response[1:] {
		monitorInfo := ReadMonitorInfo(line)
		monitorInfos = append(monitorInfos, monitorInfo)
	}

	return MultiMonitorDeviceResponse{MonitorInfo: monitorInfos}
}

func ReadMonitorInfo(data []string) MonitorInfo {
	// Ensure the slice has the correct number of fields before proceeding
	if len(data) < 20 {
		fmt.Println("Error: Insufficient data to populate MonitorInfo struct")
	}

	monitorInfo := MonitorInfo{
		Resolution:          data[0],
		LeftTop:             data[1],
		RightBottom:         data[2],
		Active:              data[3],
		Disconnected:        data[4],
		Primary:             data[5],
		Colors:              data[6],
		Frequency:           data[7],
		Orientation:         data[8],
		MaximumResolution:   data[9],
		Name:                data[10],
		Adapter:             data[11],
		DeviceID:            data[12],
		DeviceKey:           data[13],
		MonitorID:           data[14],
		ShortMonitorID:      data[15],
		MonitorKey:          data[16],
		MonitorString:       data[17],
		MonitorName:         data[18],
		MonitorSerialNumber: data[19],
	}

	return monitorInfo
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
