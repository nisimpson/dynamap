package dynamap

import (
	"fmt"
	"log"
	"time"
)

// Example demonstrates basic CRUD operations
func Example() {
	// This example shows the API without making actual AWS calls

	// Create table configuration
	table := NewTable("ecommerce-table")

	// Create a product
	product := &Product{
		ID:       "P001",
		Category: "electronics",
	}

	// Show how to marshal for put operation
	putInput, err := table.MarshalPut(product)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Put input table: %s\n", *putInput.TableName)

	// Create an order with the product
	order := &Order{
		ID:          "O001",
		PurchasedBy: "customer@example.com",
		Products:    []Product{*product},
	}

	// Show how to marshal for batch operation
	batches, err := table.MarshalBatch(order)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created %d batches for order\n", len(batches))

	// Output:
	// Put input table: ecommerce-table
	// Created 1 batches for order
}

// Example_customDelimiter demonstrates using a custom key delimiter
func Example_customDelimiter() {
	// Create table with custom delimiter
	table := NewTable("custom-table")
	table.KeyDelimiter = "|" // Use pipe instead of hash

	product := &Product{
		ID:       "P001",
		Category: "electronics",
	}

	// Marshal with custom delimiter
	relationships, err := MarshalRelationships(product, func(opts *MarshalOptions) {
		opts.KeyDelimiter = table.KeyDelimiter
	})
	if err != nil {
		log.Fatal(err)
	}

	// The keys will use the custom delimiter
	fmt.Printf("Source key: %s\n", relationships[0].Source)
	fmt.Printf("Target key: %s\n", relationships[0].Target)

	// Output:
	// Source key: product|P001
	// Target key: product|P001
}

// Example_timeToLive demonstrates setting TTL on relationships
func Example_timeToLive() {
	order := &Order{
		ID:          "O001",
		PurchasedBy: "customer@example.com",
		Products: []Product{
			{ID: "P001", Category: "electronics"},
		},
	}

	// Marshal with TTL
	relationships, err := MarshalRelationships(order, func(opts *MarshalOptions) {
		opts.TimeToLive = 30 * 24 * time.Hour // 30 days
	})

	if err != nil {
		log.Fatal(err)
	}

	// All relationships will have expiration set
	for i, rel := range relationships {
		if !rel.Expires.IsZero() {
			fmt.Printf("Relationship %d expires at: %s\n", i, rel.Expires.Format(time.RFC3339))
		}
	}

	// Output will show expiration times for all relationships
}
