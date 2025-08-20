package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/siuubhamm/distributed_kvstore/store"
)

// Global variable for key-value store
var s *store.PersistenceStore

// setRequest defines the structure of the request body,
// for the SET endpoint
type setRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// getHandler handles GET requests to /get?key=someKey.
// Example: curl "http://localhost:8080/get?key=Name"
func getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	// Getting the value from the store
	val, err := s.Get(key)
	if err != nil {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	// Write the value back in the HTTP response body
	fmt.Fprintln(w, val)
}

// setHandler handles POST requests to /set with a json body.
// Example:
//
//	curl -X POST http://localhost:8080/set \
//	     -H "Content-Type: application/json" \
//	     -d '{"key":"Name","value":"Shubham"}'
func setHandler(w http.ResponseWriter, r *http.Request) {
	var req setRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.Set(req.Key, req.Value)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "key %s set successfully", req.Key)
}

func main() {
	var err error
	s, err = store.NewPersistenceStore("persistent.json")
	if err != nil {
		log.Fatalf("failed to load persistent store: %v", err)
	}

	// Register HTTP handlers for the endpoints
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/set", setHandler)

	// Starting the server
	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
