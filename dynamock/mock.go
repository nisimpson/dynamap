package dynamock

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DynamoDBAPICall[T, U any] = func(context.Context, *T, ...func(*dynamodb.Options)) (*U, error)

// DynamoDBAPI defines the DynamoDB operations required by dynamap.
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// MockClient is a simple expectation-based mock for DynamoDB operations.
// Users can set expectations for specific operations without needing integration.
type MockClient struct {
	PutFunc            DynamoDBAPICall[dynamodb.PutItemInput, dynamodb.PutItemOutput]
	GetFunc            DynamoDBAPICall[dynamodb.GetItemInput, dynamodb.GetItemOutput]
	QueryFunc          DynamoDBAPICall[dynamodb.QueryInput, dynamodb.QueryOutput]
	BatchWriteItemFunc DynamoDBAPICall[dynamodb.BatchWriteItemInput, dynamodb.BatchWriteItemOutput]
	DeleteFunc         DynamoDBAPICall[dynamodb.DeleteItemInput, dynamodb.DeleteItemOutput]
	UpdateFunc         DynamoDBAPICall[dynamodb.UpdateItemInput, dynamodb.UpdateItemOutput]
}

// Ensure MockClient implements DynamoDBAPI
var _ DynamoDBAPI = (*MockClient)(nil)

// NewMockClient creates a new mock DynamoDB client with default configuration.
func NewMockClient(t *testing.T) *MockClient {
	return &MockClient{
		PutFunc:            defaultFunc[dynamodb.PutItemInput, dynamodb.PutItemOutput](t),
		GetFunc:            defaultFunc[dynamodb.GetItemInput, dynamodb.GetItemOutput](t),
		QueryFunc:          defaultFunc[dynamodb.QueryInput, dynamodb.QueryOutput](t),
		BatchWriteItemFunc: defaultFunc[dynamodb.BatchWriteItemInput, dynamodb.BatchWriteItemOutput](t),
		DeleteFunc:         defaultFunc[dynamodb.DeleteItemInput, dynamodb.DeleteItemOutput](t),
		UpdateFunc:         defaultFunc[dynamodb.UpdateItemInput, dynamodb.UpdateItemOutput](t),
	}
}

func defaultFunc[T, U any](t *testing.T) DynamoDBAPICall[T, U] {
	return func(ctx context.Context, params *T, optFns ...func(*dynamodb.Options)) (*U, error) {
		t.Fatal("unexpected call")
		return nil, nil
	}
}

// PutItem stores an item in the mock table.
func (m *MockClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return m.PutFunc(ctx, params, optFns...)
}

// GetItem retrieves an item from the mock table.
func (m *MockClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetFunc(ctx, params, optFns...)
}

// UpdateItem updates an item in the mock table.
func (m *MockClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	return m.UpdateFunc(ctx, params, optFns...)
}

// DeleteItem removes an item from the mock table.
func (m *MockClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return m.DeleteFunc(ctx, params, optFns...)
}

// BatchWriteItem processes batch write operations.
func (m *MockClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	return m.BatchWriteItemFunc(ctx, params, optFns...)
}

// Query performs a query operation.
func (m *MockClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.QueryFunc(ctx, params, optFns...)
}
