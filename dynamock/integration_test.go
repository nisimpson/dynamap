package dynamock

import (
	"context"
	"testing"
	"time"

	"github.com/nisimpson/dynamap"
)

// Test entities for integration testing
type TestProduct struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Price    int    `json:"price"`
}

func (p *TestProduct) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "product"
	opts.SourceID = p.ID
	opts.TargetPrefix = "product"
	opts.TargetID = p.ID
	opts.Label = "product"
	opts.RefSortKey = p.Category
	return nil
}

type TestOrder struct {
	ID       string        `json:"id"`
	Products []TestProduct `json:"-"`
}

func (o *TestOrder) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "order"
	opts.SourceID = o.ID
	opts.TargetPrefix = "order"
	opts.TargetID = o.ID
	opts.Label = "order"
	return nil
}

func (o *TestOrder) MarshalRefs(ctx *dynamap.RelationshipContext) error {
	productPtrs := make([]*TestProduct, len(o.Products))
	for i := range o.Products {
		productPtrs[i] = &o.Products[i]
	}
	ctx.AddMany("products", dynamap.SliceOf(productPtrs...))
	return nil
}

func TestNewTableManager(t *testing.T) {
	client := NewLocalClient(8000)
	tm := NewTableManager(client)

	if tm == nil {
		t.Fatal("NewTableManager returned nil")
	}

	if tm.client != client {
		t.Error("TableManager client not set correctly")
	}

	if len(tm.tables) != 0 {
		t.Error("TableManager should start with empty table list")
	}
}

func TestTableManager_GetTableNames(t *testing.T) {
	client := NewLocalClient(8000)
	tm := NewTableManager(client)

	// Initially empty
	names := tm.GetTableNames()
	if len(names) != 0 {
		t.Error("Expected empty table names initially")
	}

	// Add some table names manually (simulating table creation)
	tm.tables = append(tm.tables, "table1", "table2")

	names = tm.GetTableNames()
	if len(names) != 2 {
		t.Errorf("Expected 2 table names, got %d", len(names))
	}

	// Verify it returns a copy (modifying returned slice shouldn't affect original)
	names[0] = "modified"
	if tm.tables[0] == "modified" {
		t.Error("GetTableNames should return a copy, not the original slice")
	}
}

func TestNewTestTable(t *testing.T) {
	name1 := NewTestTable("test")
	// Add a small delay to ensure different timestamps
	time.Sleep(1 * time.Millisecond)
	name2 := NewTestTable("test")

	if name1 == name2 {
		t.Error("NewTestTable should generate unique names")
	}

	if name1 == "" || name2 == "" {
		t.Error("NewTestTable should not return empty strings")
	}
}

func TestNewSeedTestData(t *testing.T) {
	client := NewLocalClient(8000)
	seeder := NewSeedTestData(client, "test-table")

	if seeder == nil {
		t.Fatal("NewSeedTestData returned nil")
	}

	if seeder.client != client {
		t.Error("SeedTestData client not set correctly")
	}

	if seeder.tableName != "test-table" {
		t.Errorf("Expected table name test-table, got %s", seeder.tableName)
	}
}

func TestDefaultIntegrationTestConfig(t *testing.T) {
	config := DefaultIntegrationTestConfig()

	if config == nil {
		t.Fatal("DefaultIntegrationTestConfig returned nil")
	}

	if config.Port != DefaultLocalPort {
		t.Errorf("Expected port %d, got %d", DefaultLocalPort, config.Port)
	}

	if !config.SkipIfNotRunning {
		t.Error("Expected SkipIfNotRunning to be true")
	}

	if config.TablePrefix == "" {
		t.Error("Expected non-empty TablePrefix")
	}

	if config.CleanupTimeout <= 0 {
		t.Error("Expected positive CleanupTimeout")
	}
}

// TestWithIsolatedTable_Integration tests the isolated table functionality.
// This test requires DynamoDB Local to be running.
func TestWithIsolatedTable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewLocalClient(8000)
	local := &LocalDynamoDB{Client: client}

	// Check if DynamoDB Local is available
	if !local.IsAvailable(context.Background()) {
		t.Skip("DynamoDB Local not available on port 8000")
	}

	var capturedTableName string

	WithIsolatedTable(t, client, func(tableName string) {
		capturedTableName = tableName

		// Verify the table exists
		AssertTableExists(t, client, tableName)

		// Test seeding data
		seeder := NewSeedTestData(client, tableName)
		product := &TestProduct{
			ID:       "P1",
			Category: "electronics",
			Price:    299,
		}

		ctx := context.Background()
		err := seeder.SeedEntity(ctx, product)
		if err != nil {
			t.Errorf("Failed to seed entity: %v", err)
		}
	})

	// After the function returns, the table should be cleaned up
	// We can't easily test this without adding a delay, but we can verify
	// that a table name was captured
	if capturedTableName == "" {
		t.Error("Table name was not captured")
	}
}

// TestWithLocalDynamoDB_Integration tests the local DynamoDB helper.
func TestWithLocalDynamoDB_Integration(t *testing.T) {
	WithDefaultLocalDynamoDB(t, func(local *LocalDynamoDB) {
		// Test basic functionality
		ctx := context.Background()

		tables, err := local.ListTables(ctx)
		if err != nil {
			t.Errorf("Failed to list tables: %v", err)
		}

		// We don't know what tables exist, but the call should succeed
		_ = tables
	})
}

// TestRunIntegrationTest_Integration tests the integration test runner.
func TestRunIntegrationTest_Integration(t *testing.T) {
	config := DefaultIntegrationTestConfig()
	config.TablePrefix = "test-runner"

	RunIntegrationTest(t, config, func(local *LocalDynamoDB, tableName string) {
		// Verify the table exists
		AssertTableExists(t, local.Client, tableName)

		// Test seeding data with relationships
		seeder := NewSeedTestData(local.Client, tableName)
		order := &TestOrder{
			ID: "O1",
			Products: []TestProduct{
				{ID: "P1", Category: "electronics", Price: 299},
				{ID: "P2", Category: "books", Price: 19},
			},
		}

		ctx := context.Background()
		err := seeder.SeedEntityWithRefs(ctx, order)
		if err != nil {
			t.Errorf("Failed to seed entity with refs: %v", err)
		}

		// Test querying the data
		table := dynamap.NewTable(tableName)
		queryEntity := &dynamap.QueryEntity{
			Source: &TestOrder{ID: "O1"},
			Limit:  10,
		}

		queryInput, err := table.MarshalQuery(queryEntity)
		if err != nil {
			t.Errorf("Failed to marshal query: %v", err)
		}

		result, err := local.Client.Query(ctx, queryInput)
		if err != nil {
			t.Errorf("Failed to query: %v", err)
		}

		// Should have 3 items: 1 order + 2 product relationships
		if len(result.Items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result.Items))
		}
	})
}

// TestSeedTestData_Integration tests the data seeding functionality.
func TestSeedTestData_Integration(t *testing.T) {
	WithDefaultLocalDynamoDB(t, func(local *LocalDynamoDB) {
		tableName := NewTestTable("seed-test")
		ctx := context.Background()

		// Create table
		err := local.CreateDynamapTable(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		defer local.DeleteTable(ctx, tableName)

		// Test seeding
		seeder := NewSeedTestData(local.Client, tableName)

		// Seed multiple entities
		products := []dynamap.Marshaler{
			&TestProduct{ID: "P1", Category: "electronics", Price: 299},
			&TestProduct{ID: "P2", Category: "books", Price: 19},
			&TestProduct{ID: "P3", Category: "electronics", Price: 599},
		}

		err = seeder.SeedEntities(ctx, products...)
		if err != nil {
			t.Errorf("Failed to seed entities: %v", err)
		}

		// Verify data was seeded by querying
		table := dynamap.NewTable(tableName)
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
			t.Errorf("Expected 3 items, got %d", len(result.Items))
		}
	})
}

// Example of how to use the integration test helpers
func ExampleWithIsolatedTable() {
	t := &testing.T{} // In real usage, this would be passed from your test function
	client := NewDefaultLocalClient()

	WithIsolatedTable(t, client, func(tableName string) {
		// Your test code here - the table will be automatically cleaned up
		seeder := NewSeedTestData(client, tableName)
		product := &TestProduct{ID: "P1", Category: "electronics"}

		ctx := context.Background()
		_ = seeder.SeedEntity(ctx, product)

		// Test your functionality...
	})
}

// Example of how to use RunIntegrationTest
func ExampleRunIntegrationTest() {
	t := &testing.T{} // In real usage, this would be passed from your test function

	RunIntegrationTest(t, nil, func(local *LocalDynamoDB, tableName string) {
		// Your integration test code here
		// Table is automatically created and cleaned up

		seeder := NewSeedTestData(local.Client, tableName)
		// ... seed test data and run tests
		_ = seeder
	})
}
