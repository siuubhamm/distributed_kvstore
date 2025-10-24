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

var server_address = "localhost:8080"
var result_dir = "./latency_results"

const (
	num_clients      = 5
	server_path      = "./cmd/server"
	average_sleep_ms = 100
	min_ops          = 100
	max_ops          = 500
)

type LatencyRecord struct {
	Client_id  int
	Operation  string
	Latency_ms int64
}

func main() {
	flag.StringVar(&server_address, "server", "localhost:8080", "Address of the memcached-lite server")
	flag.StringVar(&result_dir, "results", "./latency_results", "Directory to save latency CSV files")
	flag.Parse()

	log.Printf("--- Automated High-Load KV Store Test (Target: %s) ---", server_address)

	if err := os.MkdirAll(result_dir, 0755); err != nil {
		log.Fatalf("Failed to create results directory %s: %v", result_dir, err)
	}

	server_executable := "server"
	if runtime.GOOS == "windows" {
		server_executable = "server.exe"
	}

	os.Remove("persistent.json")
	os.Remove(server_executable)

	var server_cmd *exec.Cmd
	if server_address == "localhost:8080" {
		log.Println("Building local server executable...")
		build_cmd := exec.Command("go", "build", "-o", server_executable, server_path)
		if output, err := build_cmd.CombinedOutput(); err != nil {
			log.Fatalf("Failed to build server: %v\nOutput: %s", err, string(output))
		}
		log.Println("Server built successfully.")

		log.Printf("Starting server from %s...", server_executable)
		server_cmd = exec.Command("./" + server_executable)
		err := server_cmd.Start()
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
		log.Println("Server process started. Waiting for it to initialize...")
		defer os.Remove(server_executable)
		time.Sleep(2 * time.Second)
	} else {
		log.Printf("Targeting remote server at %s. Skipping local build.", server_address)
	}

	var wg sync.WaitGroup
	latency_chan := make(chan LatencyRecord, num_clients*(max_ops*2))

	log.Printf("Starting %d concurrent clients, each performing %d-%d operations...", num_clients, min_ops, max_ops)
	for i := 0; i < num_clients; i++ {
		wg.Add(1)
		go run_client_tests(i, &wg, latency_chan)
	}

	wg.Wait()
	close(latency_chan)
	log.Println("All clients have finished.")

	log.Println("Processing collected latencies...")
	all_latencies := []LatencyRecord{}
	for rec := range latency_chan {
		all_latencies = append(all_latencies, rec)
	}
	save_latency_to_csv(all_latencies, fmt.Sprintf("%s/kvstore_latencies.csv", result_dir))
	log.Printf("Saved %d latency records to %s/kvstore_latencies.csv", len(all_latencies), result_dir)

	if server_cmd != nil {
		log.Println("Shutting down local server...")
		if err := server_cmd.Process.Kill(); err != nil {
			log.Fatalf("Failed to kill server process: %v", err)
		}
		log.Println("Server shut down.")
	}

	log.Println("--- Test Complete ---")
}

func run_client_tests(Client_id int, wg *sync.WaitGroup, latChan chan<- LatencyRecord) {
	defer wg.Done()

	conn, err := net.Dial("tcp", server_address)
	if err != nil {
		log.Printf("[Client %d] FAILED to connect: %v", Client_id, err)
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	num_ops := rand.Intn(max_ops-min_ops+1) + min_ops
	successCount := 0

	for i := 0; i < num_ops; i++ {
		key := fmt.Sprintf("key-%d-%d", Client_id, i)
		value := fmt.Sprintf("value-for-client-%d-op-%d", Client_id, i)
		flags := rand.Uint32()
		exptime := 60

		set_cmd := fmt.Sprintf("set %s %d %d %d\r\n%s\r\n", key, flags, exptime, len(value), value)
		start_time_set := time.Now()
		_, err := writer.WriteString(set_cmd)
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (SET Send): %v", Client_id, i, err)
			break
		}

		response, err := reader.ReadString('\n')
		latency_set := time.Since(start_time_set)
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (SET Recv): %v", Client_id, i, err)
			break
		}
		latChan <- LatencyRecord{Client_id: Client_id, Operation: "SET", Latency_ms: latency_set.Milliseconds()}

		if strings.TrimSpace(response) != "STORED" {
			log.Printf("[Client %d] FAILED on op %d: Did not get 'STORED'. Got: %s", Client_id, i, response)
			break
		}

		sleep_dur := time.Duration(rand.ExpFloat64()*average_sleep_ms) * time.Millisecond
		time.Sleep(sleep_dur)

		get_cmd := fmt.Sprintf("get %s\r\n", key)
		start_time_get := time.Now()
		_, err = writer.WriteString(get_cmd)
		if err == nil {
			err = writer.Flush()
		}
		if err != nil {
			log.Printf("[Client %d] FAILED on op %d (GET Send): %v", Client_id, i, err)
			break
		}

		value_line, err_v := reader.ReadString('\n')
		data_line, err_d := reader.ReadString('\n')
		end_line, err_e := reader.ReadString('\n')
		latency_get := time.Since(start_time_get)
		if err_v != nil || err_d != nil || err_e != nil {
			log.Printf("[Client %d] FAILED on op %d (GET Recv): V: %v, D: %v, E: %v", Client_id, i, err_v, err_d, err_e)
			break
		}
		latChan <- LatencyRecord{Client_id: Client_id, Operation: "GET", Latency_ms: latency_get.Milliseconds()}

		expectedvalue_line := fmt.Sprintf("VALUE %s %d %d", key, flags, len(value))
		if !strings.HasPrefix(value_line, expectedvalue_line) || strings.TrimSpace(data_line) != value || strings.TrimSpace(end_line) != "END" {
			log.Printf("[Client %d] FAILED on op %d: GET response was not correct.", Client_id, i)
			break
		}
		successCount++
	}

	if successCount == num_ops {
		log.Printf("[Client %d] SUCCESS: Completed all %d Set/Get operations.", Client_id, num_ops)
	} else {
		log.Printf("[Client %d] FAILED: Completed only %d of %d operations.", Client_id, successCount, num_ops)
	}
}

func save_latency_to_csv(records []LatencyRecord, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("ERROR creating CSV file %s: %v", filename, err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Client_id", "Operation", "Latency_ms"})

	for _, record := range records {
		row := []string{
			strconv.Itoa(record.Client_id),
			record.Operation,
			strconv.FormatInt(record.Latency_ms, 10),
		}
		writer.Write(row)
	}
}
