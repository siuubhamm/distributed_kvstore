package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var serverAddress = "localhost:8080"
var resultsDir = "./latency_results"

const (
	numClients      = 5
	serverPath      = "./cmd/server"
	avgSleepMillis  = 100
	minOpsPerClient = 500
	maxOpsPerClient = 1000
)

type LatencyRecord struct {
	ClientID  int
	Operation string
	LatencyMs int64
}

func main() {
	flag.StringVar(&serverAddress, "server", "localhost:8080", "Address of the memcached-lite server")
	flag.StringVar(&resultsDir, "results", "./latency_results", "Directory to save latency CSV files") // <-- Add flag for results dir
	flag.Parse()

	log.Printf("--- Automated High-Load KV Store Test (Target: %s) ---", serverAddress)

	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		log.Fatalf("Failed to create results directory %s: %v", resultsDir, err)
	}

	serverExecutable := "server"
	if runtime.GOOS == "windows" {
		serverExecutable = "server.exe"
	}

	os.Remove("persistent.json")
	os.Remove(serverExecutable)

	var serverCmd *exec.Cmd
	if serverAddress == "localhost:8080" {
		log.Println("Building local server executable...")
		buildCmd := exec.Command("go", "build", "-o", serverExecutable, serverPath)
		if output, err := buildCmd.CombinedOutput(); err != nil {
			log.Fatalf("Failed to build server: %v\nOutput: %s", err, string(output))
		}
		log.Println("Server built successfully.")

		log.Printf("Starting server from %s...", serverExecutable)
		serverCmd = exec.Command("./" + serverExecutable)
		err := serverCmd.Start()
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
		log.Println("Server process started. Waiting for it to initialize...")
		defer os.Remove(serverExecutable)
		time.Sleep(2 * time.Second)
	} else {
		log.Printf("Targeting remote server at %s. Skipping local build.", serverAddress)
	}

	var wg sync.WaitGroup
	latencyChan := make(chan LatencyRecord, numClients*(maxOpsPerClient*2))

	log.Printf("Starting %d concurrent clients, each performing %d-%d operations...", numClients, minOpsPerClient, maxOpsPerClient)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go runClientTest(i, &wg, latencyChan)
	}

	wg.Wait()
	close(latencyChan)
	log.Println("All clients have finished.")

	log.Println("Processing collected latencies...")
	allLatencies := []LatencyRecord{}
	for rec := range latencyChan {
		allLatencies = append(allLatencies, rec)
	}
	saveLatenciesToCSV(allLatencies, fmt.Sprintf("%s/kvstore_latencies.csv", resultsDir))
	log.Printf("Saved %d latency records to %s/kvstore_latencies.csv", len(allLatencies), resultsDir)

	if serverCmd != nil {
		log.Println("Shutting down local server...")
		if err := serverCmd.Process.Kill(); err != nil {
			log.Fatalf("Failed to kill server process: %v", err)
		}
		log.Println("Server shut down.")
	}

	log.Println("--- Test Complete ---")
}

func runClientTest(clientID int, wg *sync.WaitGroup, latChan chan<- LatencyRecord) {
	defer wg.Done()

	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Printf("[Client %d] FAILED to connect: %v", clientID, err)
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	numOps := rand.Intn(maxOpsPerClient-minOpsPerClient+1) + minOpsPerClient
	successCount := 0

	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("key-%d-%d", clientID, i)
		value := fmt.Sprintf("value-for-client-%d-op-%d", clientID, i)
		flags := rand.Uint32()
		exptime := 60

		setCmd := fmt.Sprintf("set %s %d %d %d\r\n%s\r\n", key, flags, exptime, len(value), value)
		startTimeSet := time.Now()
		_, err := writer.WriteString(setCmd)
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (SET Send): %v", clientID, i, err)
			break
		}

		response, err := reader.ReadString('\n')
		latencySet := time.Since(startTimeSet)
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (SET Recv): %v", clientID, i, err)
			break
		}
		latChan <- LatencyRecord{ClientID: clientID, Operation: "SET", LatencyMs: latencySet.Milliseconds()}

		if strings.TrimSpace(response) != "STORED" {
			log.Printf("[Client %d] FAILED on op %d: Did not get 'STORED'. Got: %s", clientID, i, response)
			break
		}

		sleepDuration := time.Duration(rand.ExpFloat64()*avgSleepMillis) * time.Millisecond
		time.Sleep(sleepDuration)

		getCmd := fmt.Sprintf("get %s\r\n", key)
		startTimeGet := time.Now()
		_, err = writer.WriteString(getCmd)
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (GET Send): %v", clientID, i, err)
			break
		}

		valueLine, errV := reader.ReadString('\n')
		dataLine, errD := reader.ReadString('\n')
		endLine, errE := reader.ReadString('\n')
		latencyGet := time.Since(startTimeGet)
		if errV != nil || errD != nil || errE != nil {
			log.Printf("[Client %d] FAILED on op %d (GET Recv): V: %v, D: %v, E: %v", clientID, i, errV, errD, errE)
			break
		}
		latChan <- LatencyRecord{ClientID: clientID, Operation: "GET", LatencyMs: latencyGet.Milliseconds()}

		expectedValueLine := fmt.Sprintf("VALUE %s %d %d", key, flags, len(value))
		if !strings.HasPrefix(valueLine, expectedValueLine) || strings.TrimSpace(dataLine) != value || strings.TrimSpace(endLine) != "END" {
			log.Printf("[Client %d] FAILED on op %d: GET response was not correct.", clientID, i)
			break
		}
		successCount++
	}

	if successCount == numOps {
		log.Printf("[Client %d] SUCCESS: Completed all %d Set/Get operations.", clientID, numOps)
	} else {
		log.Printf("[Client %d] FAILED: Completed only %d of %d operations.", clientID, successCount, numOps)
	}
}

func saveLatenciesToCSV(records []LatencyRecord, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("ERROR creating CSV file %s: %v", filename, err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"ClientID", "Operation", "LatencyMs"})

	for _, record := range records {
		row := []string{
			strconv.Itoa(record.ClientID),
			record.Operation,
			strconv.FormatInt(record.LatencyMs, 10),
		}
		writer.Write(row)
	}
}
