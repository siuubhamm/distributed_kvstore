package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const serverAddress = "localhost:8080"

func main() {
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to KV store server. Type 'quit' to exit.")

	go readServerResponses(conn)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}

		command := strings.ToLower(parts[0])
		switch command {
		case "set":
			if len(parts) < 3 {
				fmt.Println("ERROR: SET requires a key and a value.")
				fmt.Print("> ")
				continue
			}
			key := parts[1]
			value := strings.Join(parts[2:], " ")
			// Format as: set <key> <flags> <exptime> <bytes>\r\n<data>\r\n
			memcacheCommand := fmt.Sprintf("set %s 0 0 %d\r\n%s\r\n", key, len(value), value)
			io.WriteString(conn, memcacheCommand)
		case "get", "delete":
			if len(parts) != 2 {
				fmt.Printf("ERROR: %s requires a key.\n", strings.ToUpper(command))
				fmt.Print("> ")
				continue
			}
			fmt.Fprintf(conn, "%s\r\n", line)
		case "quit":
			fmt.Fprintln(conn, "quit")
			return
		default:
			fmt.Println("ERROR: Unknown command.")
			fmt.Print("> ")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stdin: %v", err)
	}
}

func readServerResponses(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Server closed connection.")
			} else {
				log.Printf("Error reading from server: %v", err)
			}
			os.Exit(0)
		}

		fmt.Print(response)
		// Special handling for multi-line GET response
		if strings.HasPrefix(response, "VALUE") {
			// read value line
			value, _ := reader.ReadString('\n')
			fmt.Print(value)
			// read END line
			end, _ := reader.ReadString('\n')
			fmt.Print(end)
		}
		fmt.Print("> ")
	}
}
