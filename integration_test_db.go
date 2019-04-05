package main

import (
	"context"

	v1 "github.com/asecurityteam/asset-inventory-api/pkg/handlers/v1"
)

func main() {
	db := v1.DB("localhost", 5432, "mrheinheimer", "password", "mrheinheimer")

	if err := db.SaveResource(context.Background(), "new_id"); err != nil {
		panic(err)
	}
}
