package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/fstanis/screenresolution"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const SunshineRtspPort = 48010

const mainDisplay = "PHL 346B1C"
const fakeDisplay = "Linux FHD"

// C:\Users\mail\bin\multimonitortool-x64\MultiMonitorTool.exe /disable PHL093E
// C:\Users\mail\bin\multimonitortool-x64\MultiMonitorTool.exe /enable LNX0000
const mainDisplayId = "PHL093E"
const fakeDisplayId = "LNX0000"

func main() {
	mainDisplayIsActive := false
	fakeDisplayIsActive := false
	sunshineIsStreaming := false
	for {
		mainDisplayIsActive = isMainDisplayActive()
		fakeDisplayIsActive = isFakeDisplayActive()
		sunshineIsStreaming = isSunshineStreaming()

		log.Println("Fake display is active: ", fakeDisplayIsActive)
		log.Println("Main display is active: ", mainDisplayIsActive)
		log.Println("Sunshine is streaming over RTSP: ", sunshineIsStreaming)

		if sunshineIsStreaming && (!fakeDisplayIsActive || mainDisplayIsActive) {
			log.Println("Sunshine is streaming and main display is active. Deactivate Main Display, activate fake display.")
			disableDisplay(mainDisplayId)
			enableDisplay(fakeDisplayId)

		}

		if !sunshineIsStreaming && (fakeDisplayIsActive || !mainDisplayIsActive) {
			log.Println("Sunshine is not streaming and fake display is active. Activate Main Display, deactivate Fake display")
			disableDisplay(fakeDisplayId)
			enableDisplay(mainDisplayId)
		}

		time.Sleep(10 * time.Second)
	}
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

func isSunshineStreaming() bool {
	logfile, err := readLogFile("${ProgramFiles}\\Sunshine\\config\\sunshine.log")
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

func isUltraWidescreen() bool {
	resolution := screenresolution.GetPrimary()
	if resolution == nil {
		log.Fatal("can't get resolution")
	}

	//fmt.Printf("%.2f", resolution.Height/resolution.Width)
	if resolution.Width == 2752 && resolution.Height == 1152 {
		return true
	}

	return false

}

func isFakeDisplayActive() bool {
	displayName := fakeDisplay
	return isDisplayActive(displayName)
}

func isMainDisplayActive() bool {
	displayName := mainDisplay
	return isDisplayActive(displayName)
}

func disableDisplay(monitor string) {
	err := changeDisplay("/disable", monitor)
	if err != nil {
		log.Fatal(err)
	}
}

func enableDisplay(monitor string) {
	err := changeDisplay("/enable", monitor)
	if err != nil {
		log.Fatal(err)
	}
}

func changeDisplay(command string, monitor string) error {

	// Create the command with arguments
	cmd := exec.Command("C:\\Users\\mail\\bin\\multimonitortool-x64\\MultiMonitorTool.exe", command, monitor)

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}
	return nil
}

func isDisplayActive(displayName string) bool {
	response, err := runCommandAndParseCSV("C:\\Users\\mail\\bin\\multimonitortool-x64\\MultiMonitorTool.exe", "/scomma", "dumpsys_display.csv")
	if err != nil {
		log.Fatal(err)
	}
	for i, record := range response {
		if i > 0 {
			// value 3 is active state
			// value 12 is name
			if record[3] == "Yes" && record[18] == displayName {
				return true
			}
		}
	}
	return false
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

	// Print the records (or process them as needed)
	//for _, record := range records {
	//	fmt.Println(record)
	//}
	return records, nil
}

func localPortIsAvailable(port int) bool {

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		//fmt.Fprintf(os.Stderr, "Can't listen on port %q: %s", port, err)
		return false
	}

	err = ln.Close()
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Couldn't stop listening on port %q: %s", port, err)
		return false
	}

	//fmt.Printf("TCP Port %d is available", port)
	return true
}
