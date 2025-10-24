package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	numClients      = 5
	minOpsPerClient = 50
	maxOpsPerClient = 100
	collectionName  = "kvstore_test"
	databaseID      = "kvstore-assignment"
)

var projectID string

func main() {
	log.Println("--- Firestore Performance Test ---")

	ctx := context.Background()

	// THIS LINE IS UPDATED to use the new databaseID
	client, err := firestore.NewClientWithDatabase(ctx, firestore.DetectProjectID, databaseID, option.WithCredentialsFile(""))
	if err != nil {
		log.Fatalf("Failed to create Firestore client for database '%s': %v", databaseID, err)
	}
	defer client.Close()

	log.Printf("Successfully connected to Firestore (Database: %s).", databaseID)

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
	if err := deleteCollection(ctx, client, collectionName, 100); err != nil {
		log.Printf("Failed to clean up collection: %v", err)
	}
	log.Println("Cleanup complete.")
}

func runClientTest(ctx context.Context, client *firestore.Client, clientID int, wg *sync.WaitGroup) {
	defer wg.Done()

	numOps := rand.Intn(maxOpsPerClient-minOpsPerClient+1) + minOpsPerClient
	successCount := 0

	for i := 0; i < numOps; i++ {
		key := fmt.Sprintf("key-%d-%d", clientID, i)
		value := fmt.Sprintf("value-for-client-%d-op-%d", clientID, i)

		_, err := client.Collection(collectionName).Doc(key).Set(ctx, map[string]interface{}{
			"value": value,
			"flags": rand.Uint32(),
		})
		if err != nil {
			log.Printf("[Client %d] FAILED on SET: %v", clientID, err)
			break
		}

		doc, err := client.Collection(collectionName).Doc(key).Get(ctx)
		if err != nil {
			log.Printf("[Client %d] FAILED on GET: %v", clientID, err)
			break
		}
		if !doc.Exists() {
			log.Printf("[Client %d] FAILED on GET: Doc does not exist", clientID)
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

func deleteCollection(ctx context.Context, client *firestore.Client, collPath string, batchSize int) error {
	coll := client.Collection(collPath)
	for {
		iter := coll.Limit(batchSize).Documents(ctx)
		numDeleted := 0

		batch := client.Batch()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			batch.Delete(doc.Ref)
			numDeleted++
		}

		if numDeleted == 0 {
			return nil
		}

		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
		log.Printf("Deleted %d documents from %s", numDeleted, collPath)
	}
}
