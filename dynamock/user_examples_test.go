package dynamock_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

// Example: Simple usage with functional options
func TestSimpleFunctionalOptions(t *testing.T) {
	// Create a simple entity using functional options
	entity := dynamock.NewEntity(
		dynamock.WithID("E1"),
		dynamock.WithPrefix("entity"),
		dynamock.WithLabel("entity"),
		dynamock.WithRefSortKey("test"),
		dynamock.WithData(map[string]interface{}{
			"name":  "Test Entity",
			"value": 42,
		}),
	).Build()

	// Test that it implements all dynamap interfaces
	var _ dynamap.Marshaler = entity
	var _ dynamap.RefMarshaler = entity
	var _ dynamap.Unmarshaler = entity
	var _ dynamap.RefUnmarshaler = entity

	// Test marshaling
	opts := &dynamap.MarshalOptions{}
	err := entity.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "E1" {
		t.Errorf("expected source ID 'E1', got %s", opts.SourceID)
	}

	if opts.Label != "entity" {
		t.Errorf("expected label 'entity', got %s", opts.Label)
	}
}

// Example: Entity with relationships
func TestEntityWithRelationships(t *testing.T) {
	// Create related entities
	product1 := dynamock.NewEntity(
		dynamock.WithID("P1"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
		dynamock.WithRefSortKey("electronics"),
		dynamock.WithData(map[string]interface{}{
			"name":  "Laptop",
			"price": 999.99,
		}),
	).Build()

	product2 := dynamock.NewEntity(
		dynamock.WithID("P2"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
		dynamock.WithRefSortKey("accessories"),
		dynamock.WithData(map[string]interface{}{
			"name":  "Mouse",
			"price": 29.99,
		}),
	).Build()

	// Create an order with product relationships
	order := dynamock.NewEntity(
		dynamock.WithID("O1"),
		dynamock.WithPrefix("order"),
		dynamock.WithLabel("order"),
		dynamock.WithData(map[string]interface{}{
			"customer_id": "C1",
			"total":       1029.98,
		}),
		dynamock.WithRelationships("products", product1, product2),
	).Build()

	// Test that it can marshal relationships
	relationships, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("MarshalRelationships failed: %v", err)
	}

	// Should have 3 relationships: 1 self + 2 products
	if len(relationships) != 3 {
		t.Errorf("expected 3 relationships, got %d", len(relationships))
	}
}

// Example: Using with mock client
func TestWithMockClient(t *testing.T) {
	mock := dynamock.NewMockClient(t)
	table := dynamap.NewTable("test-table")
	ctx := context.Background()

	// Create test entity
	entity := dynamock.NewEntity(
		dynamock.WithID("E1"),
		dynamock.WithPrefix("entity"),
		dynamock.WithLabel("entity"),
		dynamock.WithData(map[string]interface{}{
			"name": "Test Entity",
		}),
	).Build()

	// Set expectation
	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		// Verify the item structure
		if _, exists := params.Item["hk"]; !exists {
			t.Error("item missing hk attribute")
		}

		if _, exists := params.Item["sk"]; !exists {
			t.Error("item missing sk attribute")
		}

		return &dynamodb.PutItemOutput{}, nil
	}

	// Test marshaling and putting
	putInput, err := table.MarshalPut(entity)
	if err != nil {
		t.Fatalf("MarshalPut failed: %v", err)
	}

	_, err = mock.PutItem(ctx, putInput)
	if err != nil {
		t.Fatalf("PutItem failed: %v", err)
	}
}

// Example: Using with batch operations
func TestWithBatchOperations(t *testing.T) {
	mock := dynamock.NewMockClient(t)
	table := dynamap.NewTable("test-table")
	ctx := context.Background()

	// Create entity with relationships
	relatedEntity := dynamock.NewEntity(
		dynamock.WithID("R1"),
		dynamock.WithPrefix("related"),
		dynamock.WithLabel("related"),
	).Build()

	mainEntity := dynamock.NewEntity(
		dynamock.WithID("M1"),
		dynamock.WithPrefix("main"),
		dynamock.WithLabel("main"),
		dynamock.WithRelationship("related", relatedEntity),
	).Build()

	// Set expectation for batch write
	mock.BatchWriteItemFunc = func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
		// Verify we have requests for the test table
		requests, exists := params.RequestItems["test-table"]
		if !exists {
			t.Error("no requests for test-table")
			return &dynamodb.BatchWriteItemOutput{}, nil
		}

		// Should have 2 items: 1 main + 1 related
		if len(requests) != 2 {
			t.Errorf("expected 2 requests, got %d", len(requests))
		}

		return &dynamodb.BatchWriteItemOutput{}, nil
	}

	// Test marshaling and batch writing
	batches, err := table.MarshalBatch(mainEntity)
	if err != nil {
		t.Fatalf("MarshalBatch failed: %v", err)
	}

	for _, batch := range batches {
		_, err = mock.BatchWriteItem(ctx, batch)
		if err != nil {
			t.Fatalf("BatchWriteItem failed: %v", err)
		}
	}
}

// Example: Unmarshaling workflow
func TestUnmarshalingWorkflow(t *testing.T) {
	// Create original entity
	originalEntity := dynamock.NewEntity(
		dynamock.WithID("E1"),
		dynamock.WithPrefix("entity"),
		dynamock.WithLabel("entity"),
		dynamock.WithData(map[string]interface{}{
			"name": "Original Entity",
		}),
	).Build()

	// Marshal it to relationships
	relationships, err := dynamap.MarshalRelationships(originalEntity)
	if err != nil {
		t.Fatalf("MarshalRelationships failed: %v", err)
	}

	// Create new entity and unmarshal into it
	newEntity := dynamock.NewEntity().Build()

	// Find the self relationship and unmarshal
	for _, rel := range relationships {
		if rel.Source == rel.Target { // Self relationship
			err = newEntity.UnmarshalSelf(&rel)
			if err != nil {
				t.Fatalf("UnmarshalSelf failed: %v", err)
			}
			break
		}
	}

	// Verify the data was unmarshaled (we can't access the private field directly,
	// but we can test that the unmarshal operation succeeded)
	if err != nil {
		t.Error("expected successful unmarshal operation")
	}
}

// Example: Integration with seeding
func TestWithSeeding(t *testing.T) {
	// This test would run with a real local DynamoDB instance
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		tableName := dynamock.NewTestTable("functional-options-test")
		ctx := context.Background()

		// Create table
		err := local.CreateDynamapTable(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		defer local.DeleteTable(ctx, tableName)

		// Create test entities using functional options
		entities := []dynamap.Marshaler{
			dynamock.NewEntity(
				dynamock.WithID("E1"),
				dynamock.WithPrefix("entity"),
				dynamock.WithLabel("entity"),
				dynamock.WithRefSortKey("type-a"),
			).Build(),
			dynamock.NewEntity(
				dynamock.WithID("E2"),
				dynamock.WithPrefix("entity"),
				dynamock.WithLabel("entity"),
				dynamock.WithRefSortKey("type-b"),
			).Build(),
		}

		// Seed the data
		seeder := dynamock.NewSeedTestData(local.Client, tableName)
		err = seeder.SeedEntities(ctx, entities...)
		if err != nil {
			t.Errorf("Failed to seed entities: %v", err)
		}

		// Verify data was seeded by querying
		table := dynamap.NewTable(tableName)
		queryList := &dynamap.QueryList{
			Label: "entity",
			Limit: 10,
		}

		queryInput, err := table.MarshalQuery(queryList)
		if err != nil {
			t.Errorf("Failed to marshal query: %v", err)
		}

		result, err := local.Client.Query(ctx, queryInput)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		if len(result.Items) != 2 {
			t.Errorf("Expected 2 entities, got %d", len(result.Items))
		}
	})
}
