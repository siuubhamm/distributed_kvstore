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
	num_clients = 5
	min_ops     = 50
	max_ops     = 100
	kind_name   = "kvstore_test_entry"
	dataset_id  = "kvstore-assignment"
)

type DatastoreEntity struct {
	Value string `datastore:"value"`
	Flags int64  `datastore:"flags"`
}

func main() {
	log.Println("--- Datastore Performance Test ---")

	ctx := context.Background()

	client, err := datastore.NewClientWithDatabase(ctx, datastore.DetectProjectID, dataset_id)
	if err != nil {
		log.Fatalf("Failed to create Datastore client: %v", err)
	}
	defer client.Close()

	log.Printf("Successfully connected to Datastore (Database: %s).", dataset_id)

	var wg sync.WaitGroup
	start_time := time.Now()

	log.Printf("Starting %d concurrent clients, each performing %d-%d operations...", num_clients, min_ops, max_ops)

	for i := 0; i < num_clients; i++ {
		wg.Add(1)
		go run_client_test(ctx, client, i, &wg)
	}

	wg.Wait()
	log.Println("All clients have finished.")

	duration := time.Since(start_time)
	log.Printf("--- Test Complete ---")
	log.Printf("Total time for all operations: %v", duration)

	log.Println("Cleaning up test data...")
	if err := delete_kind(ctx, client, kind_name); err != nil {
		log.Printf("Failed to clean up kind: %v", err)
	}
	log.Println("Cleanup complete.")
}

func run_client_test(ctx context.Context, client *datastore.Client, clientID int, wg *sync.WaitGroup) {
	defer wg.Done()

	num_ops := rand.Intn(max_ops-min_ops+1) + min_ops
	success_count := 0

	for i := 0; i < num_ops; i++ {
		key_name := fmt.Sprintf("key-%d-%d", clientID, i)
		key := datastore.NameKey(kind_name, key_name, nil)

		entity := &DatastoreEntity{
			Value: fmt.Sprintf("value-for-client-%d-op-%d", clientID, i),
			Flags: rand.Int63(),
		}

		if _, err := client.Put(ctx, key, entity); err != nil {
			log.Printf("[Client %d] FAILED on SET: %v", clientID, err)
			break
		}

		var retrieved_entity DatastoreEntity
		if err := client.Get(ctx, key, &retrieved_entity); err != nil {
			log.Printf("[Client %d] FAILED on GET: %v", clientID, err)
			break
		}

		success_count++
	}

	if success_count == num_ops {
		log.Printf("[Client %d] SUCCESS: Completed all %d Set/Get operations.", clientID, num_ops)
	} else {
		log.Printf("[Client %d] FAILED: Completed only %d of %d operations.", clientID, success_count, num_ops)
	}
}

func delete_kind(ctx context.Context, client *datastore.Client, kind string) error {
	log.Println("Deleting all entities of kind 'kvstore_test_entry'...")
	for {
		q := datastore.NewQuery(kind).KeysOnly().Limit(500)
		keys, err := client.GetAll(ctx, q, nil)
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			return nil
		}

		if err := client.DeleteMulti(ctx, keys); err != nil {
			return err
		}
		log.Printf("Deleted %d entities", len(keys))
	}
}
