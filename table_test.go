package dynamap

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Mock DynamoDB client for testing
type mockDynamoDBClient struct {
	items map[string]map[string]types.AttributeValue
}

func newMockDynamoDBClient() *mockDynamoDBClient {
	return &mockDynamoDBClient{
		items: make(map[string]map[string]types.AttributeValue),
	}
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	hk := params.Item["hk"].(*types.AttributeValueMemberS).Value
	sk := params.Item["sk"].(*types.AttributeValueMemberS).Value
	key := hk + "#" + sk

	m.items[key] = params.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamoDBClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	for _, requests := range params.RequestItems {
		for _, request := range requests {
			if request.PutRequest != nil {
				hk := request.PutRequest.Item["hk"].(*types.AttributeValueMemberS).Value
				sk := request.PutRequest.Item["sk"].(*types.AttributeValueMemberS).Value
				key := hk + "#" + sk
				m.items[key] = request.PutRequest.Item
			}
		}
	}
	return &dynamodb.BatchWriteItemOutput{}, nil
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{},
	}, nil
}

func (m *mockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	hk := params.Key["hk"].(*types.AttributeValueMemberS).Value
	sk := params.Key["sk"].(*types.AttributeValueMemberS).Value
	key := hk + "#" + sk

	if item, exists := m.items[key]; exists {
		return &dynamodb.GetItemOutput{Item: item}, nil
	}

	return &dynamodb.GetItemOutput{}, nil
}

func (m *mockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	hk := params.Key["hk"].(*types.AttributeValueMemberS).Value
	sk := params.Key["sk"].(*types.AttributeValueMemberS).Value
	key := hk + "#" + sk

	delete(m.items, key)
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *mockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return &dynamodb.UpdateItemOutput{}, nil
}

// Tests for table operations

func TestTableMarshalPut(t *testing.T) {
	table := NewTable("test-table")
	product := &Product{ID: "P1", Category: "electronics"}

	t.Run("basic put", func(t *testing.T) {
		putInput, err := table.MarshalPut(product)
		if err != nil {
			t.Fatalf("Failed to marshal put: %v", err)
		}

		if *putInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *putInput.TableName)
		}

		if putInput.Item["hk"] == nil {
			t.Error("Expected hk key in item")
		}
		if putInput.Item["sk"] == nil {
			t.Error("Expected sk key in item")
		}
		if putInput.Item["label"] == nil {
			t.Error("Expected label key in item")
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		putInput, err := table.MarshalPut(product, func(opts *MarshalOptions) {
			opts.TimeToLive = time.Hour
			opts.Created = time.Now()
			opts.Updated = time.Now()
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if putInput == nil {
			t.Error("Expected non-nil put input")
		}
	})
}

func TestTableMarshalBatch(t *testing.T) {
	table := NewTable("test-table")

	t.Run("basic batch", func(t *testing.T) {
		order := &Order{
			ID:          "O1",
			PurchasedBy: "john",
			Products: []Product{
				{ID: "P1", Category: "electronics"},
				{ID: "P2", Category: "books"},
			},
		}

		batches, err := table.MarshalBatch(order)
		if err != nil {
			t.Fatalf("Failed to marshal batch: %v", err)
		}

		if len(batches) != 1 {
			t.Errorf("Expected 1 batch, got %d", len(batches))
		}

		batch := batches[0]
		requests := batch.RequestItems["test-table"]

		if len(requests) != 3 {
			t.Errorf("Expected 3 requests, got %d", len(requests))
		}
	})

	t.Run("large batch chunking", func(t *testing.T) {
		products := make([]Product, 30) // More than MaxBatchSize (25)
		for i := 0; i < 30; i++ {
			products[i] = Product{
				ID:       "P" + string(rune('1'+i)),
				Category: "electronics",
			}
		}

		order := &Order{
			ID:          "O1",
			PurchasedBy: "john",
			Products:    products,
		}

		batches, err := table.MarshalBatch(order)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(batches) < 2 {
			t.Errorf("Expected at least 2 batches for 31 relationships, got %d", len(batches))
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		order := &Order{
			ID:          "O1",
			PurchasedBy: "john",
			Products: []Product{
				{ID: "P1", Category: "electronics"},
			},
		}

		batches, err := table.MarshalBatch(order, func(opts *MarshalOptions) {
			opts.TimeToLive = time.Hour
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(batches) == 0 {
			t.Error("Expected at least one batch")
		}
	})
}

func TestTableMarshalGet(t *testing.T) {
	table := NewTable("test-table")
	product := &Product{ID: "P1", Category: "electronics"}

	t.Run("basic get", func(t *testing.T) {
		getInput, err := table.MarshalGet(product)
		if err != nil {
			t.Fatalf("Failed to marshal get: %v", err)
		}

		if *getInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *getInput.TableName)
		}

		hk := getInput.Key["hk"].(*types.AttributeValueMemberS).Value
		sk := getInput.Key["sk"].(*types.AttributeValueMemberS).Value

		if hk != "product#P1" {
			t.Errorf("Expected hk 'product#P1', got %s", hk)
		}
		if sk != "product#P1" {
			t.Errorf("Expected sk 'product#P1', got %s", sk)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		getInput, err := table.MarshalGet(product, func(opts *MarshalOptions) {
			opts.Created = time.Now()
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if getInput == nil {
			t.Error("Expected non-nil get input")
		}
	})
}

func TestTableMarshalDelete(t *testing.T) {
	table := NewTable("test-table")
	product := &Product{ID: "P1", Category: "electronics"}

	t.Run("basic delete", func(t *testing.T) {
		deleteInput, err := table.MarshalDelete(product)
		if err != nil {
			t.Fatalf("Failed to marshal delete: %v", err)
		}

		if *deleteInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *deleteInput.TableName)
		}

		hk := deleteInput.Key["hk"].(*types.AttributeValueMemberS).Value
		sk := deleteInput.Key["sk"].(*types.AttributeValueMemberS).Value

		if hk != "product#P1" {
			t.Errorf("Expected hk 'product#P1', got %s", hk)
		}
		if sk != "product#P1" {
			t.Errorf("Expected sk 'product#P1', got %s", sk)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		deleteInput, err := table.MarshalDelete(product, func(opts *MarshalOptions) {
			opts.Updated = time.Now()
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if deleteInput == nil {
			t.Error("Expected non-nil delete input")
		}
	})
}

func TestTableCustomConfiguration(t *testing.T) {
	table := NewTable("test")
	table.KeyDelimiter = "|"
	table.RefIndexName = "custom-index"
	table.PaginationTTL = time.Hour

	if table.KeyDelimiter != "|" {
		t.Errorf("Expected delimiter '|', got %s", table.KeyDelimiter)
	}
	if table.RefIndexName != "custom-index" {
		t.Errorf("Expected index name 'custom-index', got %s", table.RefIndexName)
	}
	if table.PaginationTTL != time.Hour {
		t.Errorf("Expected TTL 1h, got %v", table.PaginationTTL)
	}
}

// Test updater for MarshalUpdate tests
type testUpdater struct {
	updateFunc func(expression.UpdateBuilder) expression.UpdateBuilder
}

func (u *testUpdater) UpdateRelationship(base expression.UpdateBuilder) expression.UpdateBuilder {
	if u.updateFunc != nil {
		return u.updateFunc(base)
	}
	return base.Set(expression.Name("data.category"), expression.Value("updated"))
}

func TestTableMarshalUpdate(t *testing.T) {
	table := NewTable("test-table")
	product := &Product{ID: "P1", Category: "electronics"}

	t.Run("basic update", func(t *testing.T) {
		updater := &testUpdater{}
		updateInput, err := table.MarshalUpdate(product, updater)
		if err != nil {
			t.Fatalf("Failed to marshal update: %v", err)
		}

		if *updateInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *updateInput.TableName)
		}

		hk := updateInput.Key["hk"].(*types.AttributeValueMemberS).Value
		sk := updateInput.Key["sk"].(*types.AttributeValueMemberS).Value

		if hk != "product#P1" {
			t.Errorf("Expected hk 'product#P1', got %s", hk)
		}
		if sk != "product#P1" {
			t.Errorf("Expected sk 'product#P1', got %s", sk)
		}

		if updateInput.UpdateExpression == nil {
			t.Error("Expected update expression to be set")
		}
	})

	t.Run("nil updater", func(t *testing.T) {
		_, err := table.MarshalUpdate(product, nil)
		if err == nil {
			t.Error("Expected error with nil updater")
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		updater := &testUpdater{}
		updateInput, err := table.MarshalUpdate(product, updater, func(opts *MarshalOptions) {
			opts.Created = time.Now()
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if updateInput == nil {
			t.Error("Expected non-nil update input")
		}
	})

	t.Run("custom update function", func(t *testing.T) {
		updater := &testUpdater{
			updateFunc: func(base expression.UpdateBuilder) expression.UpdateBuilder {
				return base.Set(expression.Name("data.price"), expression.Value(100))
			},
		}

		updateInput, err := table.MarshalUpdate(product, updater)
		if err != nil {
			t.Fatalf("Failed to marshal update: %v", err)
		}

		if updateInput.UpdateExpression == nil {
			t.Error("Expected update expression to be set")
		}
	})
}
func TestDataAttribute(t *testing.T) {
	t.Run("basic data attribute", func(t *testing.T) {
		nameAttr := DataAttribute("name")
		condition := nameAttr.Equal(expression.Value("test"))
		expr, err := expression.NewBuilder().WithCondition(condition).Build()
		if err != nil {
			t.Fatalf("Failed to build expression: %v", err)
		}
		if expr.Condition() == nil {
			t.Error("Expected condition to be built")
		}
	})

	t.Run("nested data attribute", func(t *testing.T) {
		categoryAttr := DataAttribute("category")
		condition := categoryAttr.Equal(expression.Value("electronics"))
		expr, err := expression.NewBuilder().WithCondition(condition).Build()
		if err != nil {
			t.Fatalf("Failed to build expression: %v", err)
		}
		if expr.Condition() == nil {
			t.Error("Expected condition to be built")
		}
	})

	t.Run("complex data attribute", func(t *testing.T) {
		complexAttr := DataAttribute("user.profile.email")
		condition := complexAttr.Equal(expression.Value("test@example.com"))
		expr, err := expression.NewBuilder().WithCondition(condition).Build()
		if err != nil {
			t.Fatalf("Failed to build expression: %v", err)
		}
		if expr.Condition() == nil {
			t.Error("Expected condition to be built")
		}
	})
}
