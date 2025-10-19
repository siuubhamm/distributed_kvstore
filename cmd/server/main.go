package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	store "github.com/siuubhamm/distributed_kvstore/kvstore"
)

const db = "persistence.json"

func main() {
	ps, err := store.NewPersistenceStore(db)
	if err != nil {
		log.Fatalf("Failed to create persistence store: %v", err)
	}

	listener, err := net.Listen("tcp", "loalhost:8080")
	if err != nil {
		log.Fatalf("Failed to start kvstore server: %v", err)
	}

	defer listener.Close()

	log.Println("KV Store server listening on localhost:8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, ps)
	}
}

func handleConnection(conn net.Conn, ps *kvstore.PersistenceStore) {

	defer conn.Close()
	log.Printf("Client Connected: %s", conn.RemoteAddr().String())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		var response string

		switch command {
		case "SET":
			if len(parts) < 3 {
				response = "ERROR: SET command requires a key and a value \n"
			} else {
				key := parts[1]
				value := strings.Join(parts[2:], " ")

				err := ps.Set(key, value)
				if err != nil {
					response = fmt.Sprintf("ERROR : %v \n", err)
				} else {
					response = "OK \n"
				}
			}
		case "GET":
			if len(parts) != 2 {
				response = "ERROR: GET command requires a key \n"
			} else {
				key := parts[1]
				val, err := ps.Get(key)
				if err != nil {
					response = fmt.Sprintf("ERROR: %v \n", err)
				} else {
					response = fmt.Sprintf("%s \n", val)
				}
			}
		default:
			response = fmt.Sprintf("ERROR: Unknown command '%s' \n", command)
		}

		_, err := io.WriteString(conn, response)
		if err != nil {
			log.Printf("Failed to write response to client %s: %v", conn.RemoteAddr().String(), err)
			return
		}
	}

	err := scanner.Err()
	if err != nil {
		log.Printf("ERROR readign from client %s: %v", conn.RemoteAddr().String(), err)
	}
	log.Printf("Connection closed for client: %s", conn.RemoteAddr().String())
}
