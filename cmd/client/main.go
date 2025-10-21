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
	// Establish a TCP connection to the server.
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v", serverAddress, err)
	}
	defer conn.Close()

	log.Printf("Successfully connected to KV store server at %s", serverAddress)
	fmt.Println("Enter commands (e.g., SET key value, GET key, DELETE key, QUIT)")

	// Create a goroutine to read responses from the server and print them.
	// This allows us to print server responses concurrently while waiting for user input.
	go readServerResponses(conn)

	// Read commands from standard input (the user's terminal).
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			fmt.Print("> ")
			continue
		}

		// Send the user's command to the server.
		_, err := fmt.Fprintln(conn, line)
		if err != nil {
			log.Printf("Failed to send command to server: %v", err)
			break // Exit if we can't communicate with the server.
		}

		// If the user types QUIT, exit the client gracefully.
		if strings.ToUpper(strings.Fields(line)[0]) == "QUIT" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading from stdin: %v", err)
	}

	log.Println("Client shutting down.")
}

// readServerResponses continuously reads from the connection and prints to stdout.
func readServerResponses(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		response, err := reader.ReadString('\n')
		if err != nil {
			// io.EOF means the server closed the connection.
			if err == io.EOF {
				log.Println("Server closed the connection.")
			} else {
				log.Printf("Error reading from server: %v", err)
			}
			os.Exit(0)
		}

		fmt.Print(response)
		fmt.Print("> ")
	}
}
