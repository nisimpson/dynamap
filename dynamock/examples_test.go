package dynamock_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

// Example entities for testing
type Product struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Price    int    `json:"price"`
}

func (p *Product) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "product"
	opts.SourceID = p.ID
	opts.TargetPrefix = "product"
	opts.TargetID = p.ID
	opts.Label = "product"
	opts.RefSortKey = p.Category
	return nil
}

func (p *Product) UnmarshalSelf(rel *dynamap.Relationship) error {
	if data, ok := rel.Data.(*Product); ok {
		*p = *data
	}
	return nil
}

type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Products   []Product `json:"-"`
}

func (o *Order) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "order"
	opts.SourceID = o.ID
	opts.TargetPrefix = "order"
	opts.TargetID = o.ID
	opts.Label = "order"
	opts.RefSortKey = opts.Created.Format("2006-01-02")
	return nil
}

func (o *Order) MarshalRefs(ctx *dynamap.RelationshipContext) error {
	productPtrs := make([]*Product, len(o.Products))
	for i := range o.Products {
		productPtrs[i] = &o.Products[i]
	}
	ctx.AddMany("products", dynamap.SliceOf(productPtrs...))
	return nil
}

func (o *Order) UnmarshalSelf(rel *dynamap.Relationship) error {
	if data, ok := rel.Data.(*Order); ok {
		*o = *data
	}
	return nil
}

func (o *Order) UnmarshalRef(name string, id string, ref *dynamap.Relationship) error {
	if name == "products" {
		var product Product
		product.ID = id
		if err := product.UnmarshalSelf(ref); err != nil {
			return err
		}
		o.Products = append(o.Products, product)
	}
	return nil
}

// Example_basicMocking demonstrates basic usage of the mock client with dynamap.
func Example_basicMocking() {

}

// Example_errorSimulation demonstrates error injection for testing error handling.
func Example_errorSimulation() {
}

// Example_functionalOptions demonstrates using functional options.
func Example_functionalOptions() {
	// Create test entities using functional options
	product := dynamock.NewEntity(
		dynamock.WithID("P1"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
		dynamock.WithRefSortKey("electronics"),
		dynamock.WithData(map[string]interface{}{
			"id":       "P1",
			"category": "electronics",
			"price":    299,
		}),
	).Build()

	// Use the entity in tests
	_ = product
}

// Example_entityWithRelationships demonstrates creating entities with relationships.
func Example_entityWithRelationships() {
	// Create related entities
	product1 := dynamock.NewEntity(
		dynamock.WithID("P1"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
	).Build()

	product2 := dynamock.NewEntity(
		dynamock.WithID("P2"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
	).Build()

	// Create entity with relationships
	order := dynamock.NewEntity(
		dynamock.WithID("O1"),
		dynamock.WithPrefix("order"),
		dynamock.WithLabel("order"),
		dynamock.WithData(map[string]interface{}{
			"customer_id": "C1",
		}),
		dynamock.WithRelationships("products", product1, product2),
	).Build()

	// Use the entity in tests
	_ = order
}

// TestDynamapIntegration shows how to use the mock client in actual tests.
func TestDynamapIntegration(t *testing.T) {
}

// TestBatchOperations demonstrates testing batch operations.
func TestBatchOperations(t *testing.T) {
}

// TestErrorHandling demonstrates testing error scenarios.
func TestErrorHandling(t *testing.T) {
}

// TestBuilders_WithMockClient demonstrates using builders with the mock client.
func TestBuilders_WithMockClient(t *testing.T) {
	mock := dynamock.NewMockClient(t)
	table := dynamap.NewTable("test-table")
	ctx := context.Background()

	// Create test entities using functional options
	product := dynamock.NewEntity(
		dynamock.WithID("P1"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
		dynamock.WithRefSortKey("electronics"),
		dynamock.WithData(map[string]interface{}{
			"id":       "P1",
			"category": "electronics",
			"price":    299,
			"name":     "Gaming Laptop",
		}),
	).Build()

	// Set expectation for PutItem
	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		// Verify the table name
		if aws.ToString(params.TableName) != "test-table" {
			t.Errorf("expected table name test-table, got %s", aws.ToString(params.TableName))
		}

		// Verify the item has the expected structure
		if _, exists := params.Item["hk"]; !exists {
			t.Error("item missing hk attribute")
		}

		if _, exists := params.Item["sk"]; !exists {
			t.Error("item missing sk attribute")
		}

		if _, exists := params.Item["label"]; !exists {
			t.Error("item missing label attribute")
		}

		return &dynamodb.PutItemOutput{}, nil
	}

	// Test marshaling and putting the entity
	putInput, err := table.MarshalPut(product)
	if err != nil {
		t.Fatalf("MarshalPut failed: %v", err)
	}

	_, err = mock.PutItem(ctx, putInput)
	if err != nil {
		t.Fatalf("PutItem failed: %v", err)
	}
}

// TestBuilders_WithRelationships demonstrates building entities with relationships.
func TestBuilders_WithRelationships(t *testing.T) {
	mock := dynamock.NewMockClient(t)
	table := dynamap.NewTable("test-table")
	ctx := context.Background()

	// Create related entities
	product1 := dynamock.NewEntity(
		dynamock.WithID("P1"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
	).Build()

	product2 := dynamock.NewEntity(
		dynamock.WithID("P2"),
		dynamock.WithPrefix("product"),
		dynamock.WithLabel("product"),
	).Build()

	// Create an order with products
	order := dynamock.NewEntity(
		dynamock.WithID("O1"),
		dynamock.WithPrefix("order"),
		dynamock.WithLabel("order"),
		dynamock.WithData(map[string]interface{}{
			"id":          "O1",
			"customer_id": "C1",
		}),
		dynamock.WithRelationships("products", product1, product2),
	).Build()

	// Set expectation for BatchWriteItem
	mock.BatchWriteItemFunc = func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
		// Verify we have requests for the test table
		requests, exists := params.RequestItems["test-table"]
		if !exists {
			t.Error("no requests for test-table")
			return &dynamodb.BatchWriteItemOutput{}, nil
		}

		// Should have 3 items: 1 order + 2 product relationships
		if len(requests) != 3 {
			t.Errorf("expected 3 requests, got %d", len(requests))
		}

		return &dynamodb.BatchWriteItemOutput{}, nil
	}

	// Test marshaling and batch writing
	batches, err := table.MarshalBatch(order)
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

// TestBuilders_CustomConfiguration demonstrates custom entity configuration.
func TestBuilders_CustomConfiguration(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create entity with custom configuration
	entity := dynamock.NewEntity(
		dynamock.WithID("custom-id"),
		dynamock.WithPrefix("custom"),
		dynamock.WithLabel("custom-label"),
		dynamock.WithRefSortKey("custom-sort"),
		dynamock.WithCreated(fixedTime),
		dynamock.WithUpdated(fixedTime),
		dynamock.WithTimeToLive(24*time.Hour),
		dynamock.WithKeyDelimiter("|"),
		dynamock.WithLabelDelimiter("-"),
		dynamock.WithData(map[string]interface{}{
			"custom_field": "custom_value",
		}),
	).Build()

	// Test marshaling with custom configuration
	opts := &dynamap.MarshalOptions{}
	err := entity.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "custom-id" {
		t.Errorf("expected source ID 'custom-id', got %s", opts.SourceID)
	}

	if opts.KeyDelimiter != "|" {
		t.Errorf("expected key delimiter '|', got %s", opts.KeyDelimiter)
	}

	if opts.LabelDelimiter != "-" {
		t.Errorf("expected label delimiter '-', got %s", opts.LabelDelimiter)
	}

	if opts.TimeToLive != 24*time.Hour {
		t.Errorf("expected TTL 24h, got %v", opts.TimeToLive)
	}
}

// TestBuilders_IntegrationWithSeeding demonstrates using builders with data seeding.
func TestBuilders_IntegrationWithSeeding(t *testing.T) {
	// This test would run with a real local DynamoDB instance
	dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
		tableName := dynamock.NewTestTable("builder-test")
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
				dynamock.WithID("P1"),
				dynamock.WithPrefix("product"),
				dynamock.WithLabel("product"),
				dynamock.WithRefSortKey("electronics"),
			).Build(),
			dynamock.NewEntity(
				dynamock.WithID("P2"),
				dynamock.WithPrefix("product"),
				dynamock.WithLabel("product"),
				dynamock.WithRefSortKey("books"),
			).Build(),
			dynamock.NewEntity(
				dynamock.WithID("P3"),
				dynamock.WithPrefix("product"),
				dynamock.WithLabel("product"),
				dynamock.WithRefSortKey("electronics"),
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

		// Query products
		queryList := &dynamap.QueryList{
			Label: "product",
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

		if len(result.Items) != 3 {
			t.Errorf("Expected 3 products, got %d", len(result.Items))
		}
	})
}
