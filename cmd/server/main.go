package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	kvstore "github.com/siuubhamm/distributed_kvstore/kvstore"
)

const db_file = "persistent.json"

func main() {
	ps, err := kvstore.NewPersistenceStore(db_file)
	if err != nil {
		log.Fatalf("Failed to create persistence store: %v", err)
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start kvstore server: %v", err)
	}
	defer listener.Close()

	log.Println("KV Store server started on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handle_connection(conn, ps)
	}
}

func handle_connection(conn net.Conn, ps *kvstore.PersistenceStore) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	log.Printf("Client connected: %s", conn.RemoteAddr().String())

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client %s: %v", conn.RemoteAddr().String(), err)
			}
			break
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])

		switch command {
		case "set":
			handle_set(parts, reader, conn, ps)
		case "get":
			handle_get(parts, conn, ps)
		case "quit":
			return
		default:
			io.WriteString(conn, "ERROR\r\n")
		}
	}
	log.Printf("Connection closed for client: %s", conn.RemoteAddr().String())
}

func handle_set(parts []string, reader *bufio.Reader, conn net.Conn, ps *kvstore.PersistenceStore) {
	if len(parts) != 5 {
		io.WriteString(conn, "CLIENT_ERROR bad command line format\r\n")
		return
	}

	key := parts[1]
	flags, err_flags := strconv.ParseUint(parts[2], 10, 32)
	exptime, err_exptime := strconv.ParseInt(parts[3], 10, 64)
	err_count, err_bytes := strconv.Atoi(parts[4])

	if err_flags != nil || err_exptime != nil || err_bytes != nil {
		io.WriteString(conn, "CLIENT_ERROR bad command line format\r\n")
		return
	}

	value := make([]byte, err_count+2) // +2 for \r\n
	_, err := io.ReadFull(reader, value)
	if err != nil {
		log.Printf("Failed to read value data: %v", err)
		return
	}

	value_str := strings.TrimSpace(string(value))

	if err := ps.Set(key, value_str, uint32(flags), exptime); err != nil {
		io.WriteString(conn, "NOT_STORED\r\n")
		log.Printf("Failed to set key %s: %v", key, err)
	} else {
		io.WriteString(conn, "STORED\r\n")
	}
}

func handle_get(parts []string, conn net.Conn, ps *kvstore.PersistenceStore) {
	if len(parts) < 2 {
		io.WriteString(conn, "CLIENT_ERROR bad command line format\r\n")
		return
	}

	for _, key := range parts[1:] {
		item, err := ps.Get(key)
		if err == nil {
			response := fmt.Sprintf("VALUE %s %d %d\r\n%s\r\n", key, item.Flags, len(item.Value), item.Value)
			io.WriteString(conn, response)
		}
	}
	io.WriteString(conn, "END\r\n")
}
