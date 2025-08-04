package dynamock_test

import (
	"context"
	"os"
	"testing"

	"github.com/nisimpson/dynamap/dynamock"
)

// Example_seedFromJSONFile demonstrates how to seed test data from a JSON file.
func Example_seedFromJSONFile() {
	// This example shows how you would use JSON seeding in a real test
	t := &testing.T{} // In real usage, this comes from your test function parameter

	// Use the integration helper to set up DynamoDB Local
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		// Create seeder with isolated table
		dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
			seeder := dynamock.NewSeedTestData(local.Client, tableName)

			// Load test data from JSON file
			file, err := os.Open("testdata/simple.json")
			if err != nil {
				t.Fatalf("Failed to open test data file: %v", err)
			}
			defer file.Close()

			// Seed data from JSON
			count, err := seeder.SeedFromJSON(context.Background(), file)
			if err != nil {
				t.Fatalf("Failed to seed data: %v", err)
			}

			// Verify seeding results
			if count != 3 { // simple.json has 3 entities
				t.Errorf("Expected 3 entities seeded, got %d", count)
			}

			// Your test logic here - the table now contains the seeded data
			// You can query, update, or perform other operations on the seeded data
		})
	})
}

// TestSeedFromJSONFile_Simple demonstrates seeding simple entities without relationships.
func TestSeedFromJSONFile_Simple(t *testing.T) {
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
			seeder := dynamock.NewSeedTestData(local.Client, tableName)

			// Load simple test data
			file, err := os.Open("testdata/simple.json")
			if err != nil {
				t.Skipf("Test data file not found: %v", err)
			}
			defer file.Close()

			// Seed data
			count, err := seeder.SeedFromJSON(context.Background(), file)
			if err != nil {
				t.Fatalf("Failed to seed data: %v", err)
			}

			// Verify we seeded the expected number of entities
			// simple.json contains: 2 products + 1 customer = 3 entities
			if count != 3 {
				t.Errorf("Expected 3 entities seeded, got %d", count)
			}

			// In a real test, you might query the table to verify specific data
			// For example, verify that product P1 exists with correct attributes
		})
	})
}

// TestSeedFromJSONFile_Complex demonstrates seeding entities with relationships.
func TestSeedFromJSONFile_Complex(t *testing.T) {
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
			seeder := dynamock.NewSeedTestData(local.Client, tableName)

			// Load complex test data with relationships
			file, err := os.Open("testdata/complex.json")
			if err != nil {
				t.Skipf("Test data file not found: %v", err)
			}
			defer file.Close()

			// Seed data
			count, err := seeder.SeedFromJSON(context.Background(), file)
			if err != nil {
				t.Fatalf("Failed to seed data: %v", err)
			}

			// Verify we seeded the expected number of entities
			// complex.json contains: 2 orders + 1 customer = 3 entities
			if count != 3 {
				t.Errorf("Expected 3 entities seeded, got %d", count)
			}

			// In a real test, you would verify that:
			// - Order O1 exists with relationships to products P1, P2 and customer C1
			// - Order O2 exists with relationship to product P3 and customer C2
			// - Customer C2 exists with relationship to order O2
		})
	})
}

// TestSeedFromJSONFile_ErrorHandling demonstrates error handling with invalid JSON.
func TestSeedFromJSONFile_ErrorHandling(t *testing.T) {
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
			seeder := dynamock.NewSeedTestData(local.Client, tableName)

			// Try to load invalid test data
			file, err := os.Open("testdata/invalid.json")
			if err != nil {
				t.Skipf("Test data file not found: %v", err)
			}
			defer file.Close()

			// Attempt to seed data - this should fail
			_, err = seeder.SeedFromJSON(context.Background(), file)
			if err == nil {
				t.Error("Expected error when seeding invalid JSON, but got none")
			}

			// Verify the error is descriptive
			if err != nil {
				t.Logf("Got expected error: %v", err)
			}
		})
	})
}

// TestSeedFromJSONFile_RealWorldWorkflow demonstrates a complete testing workflow.
func TestSeedFromJSONFile_RealWorldWorkflow(t *testing.T) {
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
			seeder := dynamock.NewSeedTestData(local.Client, tableName)

			// Step 1: Seed initial test data
			file, err := os.Open("testdata/simple.json")
			if err != nil {
				t.Skipf("Test data file not found: %v", err)
			}
			defer file.Close()

			count, err := seeder.SeedFromJSON(context.Background(), file)
			if err != nil {
				t.Fatalf("Failed to seed initial data: %v", err)
			}

			t.Logf("Seeded %d entities from JSON file", count)

			// Step 2: Your application logic would go here
			// For example, you might:
			// - Query for products in the "electronics" category
			// - Create new orders using the seeded products
			// - Test business logic with the seeded customer data

			// Step 3: Verify results
			// You would typically query the table to verify your application
			// logic worked correctly with the seeded data

			// This demonstrates how JSON seeding provides a clean way to
			// set up complex test scenarios without writing lots of setup code
		})
	})
}
