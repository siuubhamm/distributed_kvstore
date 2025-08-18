package main

import (
	"fmt"

	"github.com/siuubhamm/distributed_kvstore/store"
)

func main() {
	store_instance := store.NewStore()

	// Setting values
	store_instance.Set("Name", "Shubham")
	store_instance.Set("Language", "Go")

	// Getting values
	val, err := store_instance.Get("Name")
	if err == nil {
		fmt.Println("Name =", val)
	}

	// Deleting values
	del_err := store_instance.Delete("Language")
	if del_err != nil {
		fmt.Println("Delte error:", del_err)
	}

	_, get_err := store_instance.Get("Langauge")
	if get_err != nil {
		fmt.Println("Get error:", get_err)
	}
}
