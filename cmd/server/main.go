package main

import (
	"fmt"
	"log"

	"github.com/siuubhamm/distributed_kvstore/store"
)

func main() {
	ps, err := store.NewPersistenceStore("persistent.json")

	if err != nil {
		log.Fatal("Failed to initialize store: ", err)
	}

	// Setting values
	ps.Set("Name", "Shubham")
	ps.Set("Language", "Go")

	// Getting values
	val, err := ps.Get("Name")
	if err == nil {
		fmt.Println("Name =", val)
	}

	// Deleting values
	del_err := ps.Delete("Language")
	if del_err != nil {
		fmt.Println("Delte error:", del_err)
	}

	_, get_err := ps.Get("Langauge")
	if get_err != nil {
		fmt.Println("Get error:", get_err)
	}
}
