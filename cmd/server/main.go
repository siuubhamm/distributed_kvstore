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

const (
	defaultFlags   = "0"
	defaultExptime = "0"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to KV store server. Type 'quit' to exit.")
	scanner := bufio.NewScanner(os.Stdin)
	serverReader := bufio.NewReader(conn)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])
		if command == "quit" {
			sendCommand(conn, "quit\r\n")
			break
		}

		err := handleCommand(command, parts, conn, serverReader)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stdin: %v", err)
	}
}

func handleCommand(command string, parts []string, conn net.Conn, serverReader *bufio.Reader) error {
	var request string

	switch command {
	case "set":
		if len(parts) < 3 {
			return fmt.Errorf("usage: set <key> <value>")
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")
		bytes := len(value)
		request = fmt.Sprintf("set %s %s %s %d\r\n%s\r\n", key, defaultFlags, defaultExptime, bytes, value)
	case "get":
		if len(parts) != 2 {
			return fmt.Errorf("usage: get <key>")
		}
		key := parts[1]
		request = fmt.Sprintf("get %s\r\n", key)
	case "delete":
		if len(parts) != 2 {
			return fmt.Errorf("usage: delete <key>")
		}
		key := parts[1]
		request = fmt.Sprintf("delete %s\r\n", key)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	if err := sendCommand(conn, request); err != nil {
		return fmt.Errorf("failed to send command to server: %w", err)
	}

	return readResponse(serverReader)
}

func sendCommand(conn net.Conn, cmd string) error {
	_, err := conn.Write([]byte(cmd))
	return err
}

func readResponse(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("connection closed by server")
			}
			return err
		}

		fmt.Print(line)

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "END" || trimmedLine == "STORED" || trimmedLine == "NOT_STORED" || trimmedLine == "DELETED" || trimmedLine == "NOT_FOUND" || strings.HasPrefix(trimmedLine, "CLIENT_ERROR") || strings.HasPrefix(trimmedLine, "ERROR") {
			break
		}
	}
	return nil
}
