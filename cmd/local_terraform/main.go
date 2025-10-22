package main

import (
	"bufio"
	"flag" // <-- IMPORTANT: Imports the flag package
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Make serverAddress a variable (not a const) so the flag can change it
var serverAddress = "localhost:8080"

const (
	numClients      = 5
	serverPath      = "./cmd/server"
	avgSleepMillis  = 100 // Lowering sleep for higher throughput
	minOpsPerClient = 50
	maxOpsPerClient = 100
)

func main() {
	// Define and parse the -server flag from the command line
	flag.StringVar(&serverAddress, "server", "localhost:8080", "Address of the memcached-lite server")
	flag.Parse()

	log.Printf("--- Automated High-Load KV Store Test (Target: %s) ---", serverAddress)

	serverExecutable := "server"
	if runtime.GOOS == "windows" {
		serverExecutable = "server.exe"
	}

	os.Remove("persistent.json")
	os.Remove(serverExecutable)

	var serverCmd *exec.Cmd
	// --- THIS IS THE NEW LOGIC ---
	// If the server address is the default, build and run it locally.
	// If it's a remote address, skip all this.
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
	// --- END NEW LOGIC ---

	var wg sync.WaitGroup

	log.Printf("Starting %d concurrent clients, each performing %d-%d operations...", numClients, minOpsPerClient, maxOpsPerClient)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go runClientTest(i, &wg)
	}

	wg.Wait()
	log.Println("All clients have finished.")

	// --- NEW LOGIC ---
	// Only shut down the server if we are the ones who started it.
	if serverCmd != nil {
		log.Println("Shutting down local server...")
		if err := serverCmd.Process.Kill(); err != nil {
			log.Fatalf("Failed to kill server process: %v", err)
		}
		log.Println("Server shut down.")
	}
	// --- END NEW LOGIC ---

	log.Println("--- Test Complete ---")
}

func runClientTest(clientID int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Connect to the serverAddress variable (which is set by the flag)
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Printf("[Client %d] FAILED to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	numOps := rand.Intn(maxOpsPerClient-minOpsPerClient+1) + minOpsPerClient
	successCount := 0

	for i := 0; i < numOps; i++ {
		// Each operation gets a unique key
		key := fmt.Sprintf("key-%d-%d", clientID, i)
		value := fmt.Sprintf("value-for-client-%d-op-%d", clientID, i)
		flags := rand.Uint32()
		exptime := 60 // Set a long expiry time

		// 1. SET the value
		setCmd := fmt.Sprintf("set %s %d %d %d\r\n%s\r\n", key, flags, exptime, len(value), value)
		io.WriteString(conn, setCmd)
		response, _ := bufio.NewReader(conn).ReadString('\n')
		if strings.TrimSpace(response) != "STORED" {
			log.Printf("[Client %d] FAILED on op %d: Did not get 'STORED'. Got: %s", clientID, i, response)
			break // Stop this client on failure
		}

		sleepDuration := time.Duration(rand.ExpFloat64()*avgSleepMillis) * time.Millisecond
		time.Sleep(sleepDuration)

		// 2. GET the value back
		getCmd := fmt.Sprintf("get %s\r\n", key)
		io.WriteString(conn, getCmd)

		reader := bufio.NewReader(conn)
		valueLine, _ := reader.ReadString('\n')
		dataLine, _ := reader.ReadString('\n')
		endLine, _ := reader.ReadString('\n')

		expectedValueLine := fmt.Sprintf("VALUE %s %d %d", key, flags, len(value))
		if !strings.HasPrefix(valueLine, expectedValueLine) || strings.TrimSpace(dataLine) != value || strings.TrimSpace(endLine) != "END" {
			log.Printf("[Client %d] FAILED on op %d: GET response was not correct.", clientID, i)
			break // Stop this client on failure
		}
		successCount++
	}

	if successCount == numOps {
		log.Printf("[Client %d] SUCCESS: Completed all %d Set/Get operations.", clientID, numOps)
	} else {
		log.Printf("[Client %d] FAILED: Completed only %d of %d operations.", clientID, successCount, numOps)
	}
}
