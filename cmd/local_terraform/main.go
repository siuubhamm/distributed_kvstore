package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	serverAddress  = "localhost:8080"
	numClients     = 5
	serverPath     = "./cmd/server"
	avgSleepMillis = 500 // The average sleep time for our exponential distribution
)

func main() {
	log.Println("--- Automated KV Store Test ---")

	os.Remove("persistent.json")

	log.Printf("Starting server from %s...", serverPath)
	serverCmd := exec.Command("go", "run", serverPath)
	err := serverCmd.Start()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Println("Server process started. Waiting for it to initialize...")

	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	log.Printf("Starting %d concurrent clients...", numClients)
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go runClientTest(i, &wg)
	}

	wg.Wait()
	log.Println("All clients have finished.")

	log.Println("Shutting down server...")
	if err := serverCmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to kill server process: %v", err)
	}
	log.Println("Server shut down.")

	log.Println("--- Test Complete ---")
}

func runClientTest(clientID int, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Printf("[Client %d] FAILED to connect: %v", clientID, err)
		return
	}
	defer conn.Close()

	key := fmt.Sprintf("key-%d", clientID)
	value := fmt.Sprintf("value-for-client-%d", clientID)

	// Generate random flags and a short, random expiry time (0-9 seconds)
	flags := rand.Uint32()
	exptime := rand.Intn(10)

	// 1. SET the value with the random parameters
	setCmd := fmt.Sprintf("set %s %d %d %d\r\n%s\r\n", key, flags, exptime, len(value), value)
	io.WriteString(conn, setCmd)
	response, _ := bufio.NewReader(conn).ReadString('\n')
	if strings.TrimSpace(response) != "STORED" {
		log.Printf("[Client %d] FAILED: Did not get 'STORED' response for SET. Got: %s", clientID, response)
		return
	}

	// Introduce a variable sleep time between requests.
	sleepDuration := time.Duration(rand.ExpFloat64()*avgSleepMillis) * time.Millisecond
	time.Sleep(sleepDuration)

	// 2. GET the value back
	getCmd := fmt.Sprintf("get %s\r\n", key)
	io.WriteString(conn, getCmd)

	reader := bufio.NewReader(conn)
	valueLine, _ := reader.ReadString('\n')
	dataLine, _ := reader.ReadString('\n')
	endLine, _ := reader.ReadString('\n')

	// The expected response must now match the random flags we sent.
	expectedValueLine := fmt.Sprintf("VALUE %s %d %d", key, flags, len(value))
	if !strings.HasPrefix(valueLine, expectedValueLine) || strings.TrimSpace(dataLine) != value || strings.TrimSpace(endLine) != "END" {
		log.Printf("[Client %d] FAILED: GET response was not correct. Sent (Flags: %d, Expiry: %d).", clientID, flags, exptime)
		return
	}

	log.Printf("[Client %d] SUCCESS: Set/Get verified for key '%s' (Flags: %d, Expiry: %ds, Sleep: %v)", clientID, key, flags, exptime, sleepDuration)
}
