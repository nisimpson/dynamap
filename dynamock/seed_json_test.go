package dynamock

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSeedFromJSON_ParseSimpleEntity(t *testing.T) {
	// Create a minimal SeedTestData for testing parsing logic
	seedData := &SeedTestData{
		client:    nil, // We won't use the client in these tests
		tableName: "test-table",
	}

	// Simple JSON with single entity, no relationships
	jsonData := `[
		{
			"type": "product",
			"id": "P1",
			"attributes": {
				"name": "Laptop",
				"category": "electronics",
				"price": 999
			}
		}
	]`

	// Parse JSON document
	var document JSONAPIDocument
	err := parseJSONDocument(strings.NewReader(jsonData), &document)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Test conversion to entity
	if len(document) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(document))
	}

	resource := document[0]
	entity, err := seedData.convertResourceToEntity(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource: %v", err)
	}

	// Verify entity properties
	if entity.opts.SourcePrefix != "product" {
		t.Errorf("Expected sourcePrefix 'product', got %s", entity.opts.SourcePrefix)
	}
	if entity.opts.SourceID != "P1" {
		t.Errorf("Expected sourceID 'P1', got %s", entity.opts.SourceID)
	}
	if entity.opts.Label != "product" {
		t.Errorf("Expected label 'product', got %s", entity.opts.Label)
	}

	// Verify attributes
	data, ok := entity.data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map[string]interface{}")
	}
	if data["name"] != "Laptop" {
		t.Errorf("Expected name 'Laptop', got %v", data["name"])
	}
	if data["category"] != "electronics" {
		t.Errorf("Expected category 'electronics', got %v", data["category"])
	}
}

func TestSeedFromJSON_ParseEntityWithRelationships(t *testing.T) {
	seedData := &SeedTestData{
		client:    nil,
		tableName: "test-table",
	}

	// JSON with entity that has relationships
	jsonData := `[
		{
			"type": "order",
			"id": "O1",
			"attributes": {
				"customer_id": "C1",
				"total": 1024
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
				}
			}
		}
	]`

	// Parse JSON document
	var document JSONAPIDocument
	err := parseJSONDocument(strings.NewReader(jsonData), &document)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Test conversion to entity
	resource := document[0]
	entity, err := seedData.convertResourceToEntity(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource: %v", err)
	}

	// Verify base entity
	if entity.opts.SourcePrefix != "order" {
		t.Errorf("Expected sourcePrefix 'order', got %s", entity.opts.SourcePrefix)
	}
	if entity.opts.SourceID != "O1" {
		t.Errorf("Expected sourceID 'O1', got %s", entity.opts.SourceID)
	}

	// Verify relationships
	products, exists := entity.relationships["products"]
	if !exists {
		t.Fatal("Expected 'products' relationship")
	}
	if len(products) != 2 {
		t.Errorf("Expected 2 product relationships, got %d", len(products))
	}
}

func TestSeedFromJSON_ParseSingleRelationship(t *testing.T) {
	seedData := &SeedTestData{
		client:    nil,
		tableName: "test-table",
	}

	// JSON with single relationship (not array)
	jsonData := `[
		{
			"type": "order",
			"id": "O1",
			"attributes": {
				"customer_id": "C1"
			},
			"relationships": {
				"customer": {
					"data": {
						"type": "customer",
						"id": "C1"
					}
				}
			}
		}
	]`

	// Parse and convert
	var document JSONAPIDocument
	err := parseJSONDocument(strings.NewReader(jsonData), &document)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	entity, err := seedData.convertResourceToEntity(document[0])
	if err != nil {
		t.Fatalf("Failed to convert resource: %v", err)
	}

	// Verify single relationship
	customers, exists := entity.relationships["customer"]
	if !exists {
		t.Fatal("Expected 'customer' relationship")
	}
	if len(customers) != 1 {
		t.Errorf("Expected 1 customer relationship, got %d", len(customers))
	}
}

func TestSeedFromJSON_ParseMultipleEntities(t *testing.T) {
	seedData := &SeedTestData{
		client:    nil,
		tableName: "test-table",
	}

	// JSON with multiple entities
	jsonData := `[
		{
			"type": "product",
			"id": "P1",
			"attributes": {
				"name": "Laptop"
			}
		},
		{
			"type": "product",
			"id": "P2",
			"attributes": {
				"name": "Mouse"
			}
		},
		{
			"type": "customer",
			"id": "C1",
			"attributes": {
				"name": "John Doe"
			}
		}
	]`

	// Parse JSON document
	var document JSONAPIDocument
	err := parseJSONDocument(strings.NewReader(jsonData), &document)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify we have 3 resources
	if len(document) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(document))
	}

	// Convert all resources
	for i, resource := range document {
		entity, err := seedData.convertResourceToEntity(resource)
		if err != nil {
			t.Fatalf("Failed to convert resource %d: %v", i, err)
		}

		// Basic validation
		if entity.opts.SourcePrefix == "" {
			t.Errorf("Resource %d missing sourcePrefix", i)
		}
		if entity.opts.SourceID == "" {
			t.Errorf("Resource %d missing sourceID", i)
		}
	}
}

func TestSeedFromJSON_ErrorCases(t *testing.T) {
	seedData := &SeedTestData{
		client:    nil,
		tableName: "test-table",
	}

	testCases := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "Invalid JSON",
			jsonData: `{invalid json}`,
			wantErr:  true,
		},
		{
			name: "Missing type field",
			jsonData: `[
				{
					"id": "P1",
					"attributes": {"name": "Product"}
				}
			]`,
			wantErr: true,
		},
		{
			name: "Missing id field",
			jsonData: `[
				{
					"type": "product",
					"attributes": {"name": "Product"}
				}
			]`,
			wantErr: true,
		},
		{
			name: "Invalid relationship data",
			jsonData: `[
				{
					"type": "order",
					"id": "O1",
					"relationships": {
						"products": {
							"data": "invalid"
						}
					}
				}
			]`,
			wantErr: true,
		},
		{
			name: "Missing relationship type",
			jsonData: `[
				{
					"type": "order",
					"id": "O1",
					"relationships": {
						"products": {
							"data": [
								{
									"id": "P1"
								}
							]
						}
					}
				}
			]`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var document JSONAPIDocument
			err := parseJSONDocument(strings.NewReader(tc.jsonData), &document)

			if tc.wantErr && err == nil {
				// Try conversion if parsing succeeded
				if len(document) > 0 {
					_, err = seedData.convertResourceToEntity(document[0])
				}
			}

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSeedFromJSON_NullRelationships(t *testing.T) {
	seedData := &SeedTestData{
		client:    nil,
		tableName: "test-table",
	}

	// JSON with null relationship data
	jsonData := `[
		{
			"type": "order",
			"id": "O1",
			"attributes": {
				"customer_id": "C1"
			},
			"relationships": {
				"products": {
					"data": null
				}
			}
		}
	]`

	// Parse and convert
	var document JSONAPIDocument
	err := parseJSONDocument(strings.NewReader(jsonData), &document)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	entity, err := seedData.convertResourceToEntity(document[0])
	if err != nil {
		t.Fatalf("Failed to convert resource: %v", err)
	}

	// Verify null relationships are handled
	products, exists := entity.relationships["products"]
	if exists && len(products) > 0 {
		t.Error("Expected no products for null relationship data")
	}
}

// TestSeedFromJSON_Integration tests the full integration with a real client
func TestSeedFromJSON_Integration(t *testing.T) {
	// Skip if DynamoDB Local is not available
	WithDefaultLocalDynamoDB(t, func(local *LocalDynamoDB) {
		// Use WithIsolatedTable to handle table creation/deletion
		WithIsolatedTable(t, local.Client, func(isolatedTableName string) {
			// Create seeder
			seedData := NewSeedTestData(local.Client, isolatedTableName)

			// Test JSON data
			jsonData := `[
				{
					"type": "product",
					"id": "P1",
					"attributes": {
						"name": "Test Product",
						"category": "test"
					}
				}
			]`

			// Seed from JSON
			count, err := seedData.SeedFromJSON(context.Background(), strings.NewReader(jsonData))
			if err != nil {
				t.Fatalf("SeedFromJSON failed: %v", err)
			}

			if count != 1 {
				t.Errorf("Expected count 1, got %d", count)
			}

			// Verify data was seeded (basic check)
			// In a real test, you might query the table to verify the data
		})
	})
}

// Helper function to parse JSON documents
func parseJSONDocument(r *strings.Reader, document *JSONAPIDocument) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(document)
}
