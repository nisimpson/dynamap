// Package dynamock provides testing utilities for the dynamap library.
//
// This package includes:
//   - Expectation-based mock DynamoDB client for unit testing
//   - Local DynamoDB integration utilities
//   - Generic test data builders with fluent and functional APIs
//   - Test data seeding helpers
//   - Integration test utilities with automatic cleanup
//
// # Mock Client
//
// The MockClient provides an expectation-based mock implementation where you set
// expectations for specific operations:
//
//	mock := dynamock.NewMockClient(t)
//
//	// Set expectation for PutItem
//	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
//		// Verify the operation parameters
//		return &dynamodb.PutItemOutput{}, nil
//	}
//
//	// Use mock in your tests
//	table := dynamap.NewTable("test-table")
//	putInput, _ := table.MarshalPut(entity)
//	_, err := mock.PutItem(ctx, putInput)
//
// # Generic Test Data Builders
//
// The package provides both fluent builders and functional options for creating test entities:
//
//	// Fluent builder API
//	entity := dynamock.NewEntity().
//		WithID("E1").
//		WithPrefix("entity").
//		WithLabel("entity").
//		WithRefSortKey("sort-key").
//		WithData(map[string]interface{}{
//			"field": "value",
//		}).
//		Build()
//
//	// Functional options API
//	entity := dynamock.NewEntity(
//		dynamock.WithID("E1"),
//		dynamock.WithPrefix("entity"),
//		dynamock.WithLabel("entity"),
//		dynamock.WithData(map[string]interface{}{
//			"field": "value",
//		}),
//	).Build()
//
// # Domain-Specific Builders (Optional)
//
// For common e-commerce entities, import the presets subpackage:
//
//	import "github.com/nisimpson/dynamap/dynamock/presets"
//
//	// Use domain-specific builders
//	product := presets.NewProduct().
//		WithID("P1").
//		WithCategory("electronics").
//		WithPrice(299).
//		Build()
//
//	customer := presets.NewCustomer().
//		WithID("C1").
//		WithEmail("test@example.com").
//		Premium().
//		Build()
//
//	// Quick helpers
//	order := presets.QuickOrder("O1", "C1")
//
// # Local DynamoDB
//
// For integration testing, the package provides utilities to work with
// local DynamoDB instances:
//
//	// Simple client creation
//	client := dynamock.NewLocalClient(8000)
//
//	// Full local DynamoDB instance with utilities
//	local := dynamock.NewLocalDynamoDB(8000)
//	if local.IsAvailable(ctx) {
//		tableName := "test-table"
//		err := local.CreateDynamapTable(ctx, tableName)
//		// ... run tests
//		err = local.DeleteTable(ctx, tableName)
//	}
//
// # Integration Test Helpers
//
// The package provides several helpers for integration testing:
//
//	// Isolated table that's automatically cleaned up
//	dynamock.WithIsolatedTable(t, client, func(tableName string) {
//		// Your test code here
//	})
//
//	// Full integration test runner
//	dynamock.RunIntegrationTest(t, nil, func(local *LocalDynamoDB, tableName string) {
//		// Your integration test code here
//	})
//
// # Test Data Seeding
//
// Easily seed test data into tables:
//
//	seeder := dynamock.NewSeedTestData(client, tableName)
//
//	// Seed a single entity
//	err := seeder.SeedEntity(ctx, entity)
//
//	// Seed an entity with all its relationships
//	err := seeder.SeedEntityWithRefs(ctx, order)
//
//	// Seed multiple entities
//	err := seeder.SeedEntities(ctx, entity1, entity2, entity3)
//
// # Table Management
//
// Automatic table lifecycle management for tests:
//
//	tm := dynamock.NewTableManager(client)
//
//	// Create tables (automatically tracked)
//	err := tm.CreateTestTable(ctx, "table1")
//	err := tm.CreateTestTable(ctx, "table2")
//
//	// Cleanup all created tables
//	defer tm.Cleanup(ctx)
//
// # Complete Example
//
// Here's a complete example using generic builders:
//
//	func TestCompleteExample(t *testing.T) {
//		// Create test entities using generic builders
//		product := dynamock.NewEntity().
//			WithID("P1").
//			WithPrefix("product").
//			WithLabel("product").
//			WithRefSortKey("electronics").
//			Build()
//
//		order := dynamock.NewEntity().
//			WithID("O1").
//			WithPrefix("order").
//			WithLabel("order").
//			WithRelationship("products", product).
//			Build()
//
//		// Test with mock client
//		mock := dynamock.NewMockClient(t)
//		mock.BatchWriteItemFunc = func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
//			// Verify batch write parameters
//			return &dynamodb.BatchWriteItemOutput{}, nil
//		}
//
//		table := dynamap.NewTable("test-table")
//		batches, _ := table.MarshalBatch(order)
//		for _, batch := range batches {
//			_, _ = mock.BatchWriteItem(context.Background(), batch)
//		}
//	}
//
// See the DESIGN.md file for detailed design documentation and examples.
package dynamock
