package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
)

const (
	numClients      = 5
	minOpsPerClient = 50
	maxOpsPerClient = 100
	kindName        = "kvstore_test_entry"
	// THIS IS THE NEW, CRITICAL LINE
	databaseID = "kvstore-assignment"
)

// DatastoreEntity is the struct we will save.
type DatastoreEntity struct {
	Value string `datastore:"value"`
	// CHANGED: Datastore doesn't support uint32, so we use int64
	Flags int64 `datastore:"flags"`
}

func main() {
	log.Println("--- Datastore Performance Test ---")

	ctx := context.Background()

	// Use the Datastore client, AND specify the databaseID
	client, err := datastore.NewClientWithDatabase(ctx, datastore.DetectProjectID, databaseID)
	if err != nil {
		log.Fatalf("Failed to create Datastore client: %v", err)
	}
	defer client.Close()

	log.Printf("Successfully connected to Datastore (Database: %s).", databaseID)

	var wg sync.WaitGroup
	startTime := time.Now()

	log.Printf("Starting %d concurrent clients, each performing %d-%d operations...", numClients, minOpsPerClient, maxOpsPerClient)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go runClientTest(ctx, client, i, &wg)
	}

	wg.Wait()
	log.Println("All clients have finished.")

	duration := time.Since(startTime)
	log.Printf("--- Test Complete ---")
	log.Printf("Total time for all operations: %v", duration)

	log.Println("Cleaning up test data...")
	if err := deleteKind(ctx, client, kindName); err != nil {
		log.Printf("Failed to clean up kind: %v", err)
	}
	log.Println("Cleanup complete.")
}

func runClientTest(ctx context.Context, client *datastore.Client, clientID int, wg *sync.WaitGroup) {
	defer wg.Done()

	numOps := rand.Intn(maxOpsPerClient-minOpsPerClient+1) + minOpsPerClient
	successCount := 0

	for i := 0; i < numOps; i++ {
		keyName := fmt.Sprintf("key-%d-%d", clientID, i)
		key := datastore.NameKey(kindName, keyName, nil)

		entity := &DatastoreEntity{
			Value: fmt.Sprintf("value-for-client-%d-op-%d", clientID, i),
			// CHANGED: Use rand.Int63() to generate an int64
			Flags: rand.Int63(),
		}

		// 1. SET the value (using "Put")
		if _, err := client.Put(ctx, key, entity); err != nil {
			log.Printf("[Client %d] FAILED on SET: %v", clientID, err)
			break
		}

		// 2. GET the value
		var retrievedEntity DatastoreEntity
		if err := client.Get(ctx, key, &retrievedEntity); err != nil {
			log.Printf("[Client %d] FAILED on GET: %v", clientID, err)
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

// deleteKind is the Datastore way of "deleting a collection"
func deleteKind(ctx context.Context, client *datastore.Client, kind string) error {
	log.Println("Deleting all entities of kind 'kvstore_test_entry'...")
	for {
		q := datastore.NewQuery(kind).KeysOnly().Limit(500)
		keys, err := client.GetAll(ctx, q, nil)
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			// Nothing left to delete
			return nil
		}

		if err := client.DeleteMulti(ctx, keys); err != nil {
			return err
		}
		log.Printf("Deleted %d entities", len(keys))
	}
}
