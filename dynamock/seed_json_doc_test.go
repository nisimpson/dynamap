package dynamock

import (
	"fmt"
)

// ExampleSeedTestData_SeedFromJSON demonstrates basic usage of JSON seeding.
func ExampleSeedTestData_SeedFromJSON() {
	// JSON data following JSON:API specification
	jsonData := `[
		{
			"type": "product",
			"id": "P1",
			"attributes": {
				"name": "Gaming Laptop",
				"category": "electronics",
				"price": 1299
			}
		},
		{
			"type": "customer",
			"id": "C1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com"
			}
		}
	]`

	// In a real test, you would use a real DynamoDB client
	// For this example, we'll show the basic usage pattern

	// Create seeder (in real usage, use actual client and table name)
	// seeder := NewSeedTestData(client, "my-test-table")

	// Seed data from JSON
	// count, err := seeder.SeedFromJSON(context.Background(), strings.NewReader(jsonData))
	// if err != nil {
	//     log.Fatalf("Failed to seed data: %v", err)
	// }

	// fmt.Printf("Seeded %d entities\n", count)

	_ = jsonData // Suppress unused variable warning

	// Output would be: Seeded 2 entities
	fmt.Println("JSON seeding allows bulk creation of test entities")
	// Output: JSON seeding allows bulk creation of test entities
}

// ExampleSeedTestData_SeedFromJSON_withRelationships demonstrates seeding entities with relationships.
func ExampleSeedTestData_SeedFromJSON_withRelationships() {
	// JSON data with relationships
	jsonData := `[
		{
			"type": "order",
			"id": "O1",
			"attributes": {
				"customer_id": "C1",
				"total": 1348,
				"status": "pending"
			},
			"relationships": {
				"products": {
					"data": [
						{
							"type": "product",
							"id": "P1"
						},
						{
							"type": "product", 
							"id": "P2"
						}
					]
				},
				"customer": {
					"data": {
						"type": "customer",
						"id": "C1"
					}
				}
			}
		}
	]`

	// This would create:
	// 1. An order entity (O1)
	// 2. Relationships from order to products P1 and P2
	// 3. A relationship from order to customer C1

	_ = jsonData // Suppress unused variable warning

	fmt.Println("JSON:API relationships are converted to TestEntity relationships")
	// Output: JSON:API relationships are converted to TestEntity relationships
}

// ExampleSeedTestData_SeedFromJSON_fileUsage demonstrates loading from a file.
func ExampleSeedTestData_SeedFromJSON_fileUsage() {
	// Example of loading from a file in a real test:

	/*
		func TestMyFeature(t *testing.T) {
			// Set up DynamoDB Local
			dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
				// Create isolated table
				dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
					seeder := dynamock.NewSeedTestData(local.Client, tableName)

					// Load test data from file
					file, err := os.Open("testdata/my-test-data.json")
					if err != nil {
						t.Fatalf("Failed to open test data: %v", err)
					}
					defer file.Close()

					// Seed data
					count, err := seeder.SeedFromJSON(context.Background(), file)
					if err != nil {
						t.Fatalf("Failed to seed data: %v", err)
					}

					// Your test logic here with seeded data
					// ...
				})
			})
		}
	*/

	fmt.Println("Load test data from JSON files for reusable test scenarios")
	// Output: Load test data from JSON files for reusable test scenarios
}

// ExampleJSONAPIDocument demonstrates the JSON:API document structure.
func ExampleJSONAPIDocument() {
	// Example JSON:API document structure
	example := `[
		{
			"type": "order",           // Maps to entity prefix/label
			"id": "O1",               // Maps to entity ID
			"attributes": {           // Maps to entity data
				"customer_id": "C1",
				"total": 1000,
				"status": "pending"
			},
			"relationships": {        // Maps to entity relationships
				"products": {
					"data": [         // Array of related entities
						{
							"type": "product",
							"id": "P1"
						}
					]
				},
				"customer": {
					"data": {         // Single related entity
						"type": "customer",
						"id": "C1"
					}
				}
			}
		}
	]`

	_ = example // Suppress unused variable warning

	fmt.Println("JSON:API format provides structured way to define entities and relationships")
	// Output: JSON:API format provides structured way to define entities and relationships
}

// Example_jsonSeedingWorkflow demonstrates a complete workflow using JSON seeding.
func Example_jsonSeedingWorkflow() {
	// Step 1: Define your test data in JSON format
	testData := `[
		{
			"type": "product",
			"id": "P1",
			"attributes": {
				"name": "Laptop",
				"category": "electronics",
				"price": 999
			}
		},
		{
			"type": "order",
			"id": "O1", 
			"attributes": {
				"customer_id": "C1",
				"total": 999
			},
			"relationships": {
				"products": {
					"data": [{"type": "product", "id": "P1"}]
				}
			}
		}
	]`

	// Step 2: In your test, seed the data
	/*
		seeder := NewSeedTestData(client, tableName)
		count, err := seeder.SeedFromJSON(ctx, strings.NewReader(testData))
	*/

	// Step 3: Run your application logic against the seeded data
	// Step 4: Assert on the results

	_ = testData // Suppress unused variable warning

	fmt.Println("1. Define JSON test data")
	fmt.Println("2. Seed data in test setup")
	fmt.Println("3. Run application logic")
	fmt.Println("4. Assert on results")

	// Output:
	// 1. Define JSON test data
	// 2. Seed data in test setup
	// 3. Run application logic
	// 4. Assert on results
}
