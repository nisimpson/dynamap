package dynamock

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestNewMockClient(t *testing.T) {
	mock := NewMockClient(t)

	if mock == nil {
		t.Fatal("NewMockClient returned nil")
	}

	if mock.PutFunc == nil {
		t.Error("PutFunc not initialized")
	}

	if mock.GetFunc == nil {
		t.Error("GetFunc not initialized")
	}

	if mock.QueryFunc == nil {
		t.Error("QueryFunc not initialized")
	}

	if mock.BatchWriteItemFunc == nil {
		t.Error("BatchWriteItemFunc not initialized")
	}

	if mock.DeleteFunc == nil {
		t.Error("DeleteFunc not initialized")
	}

	if mock.UpdateFunc == nil {
		t.Error("UpdateFunc not initialized")
	}
}

func TestMockClient_PutItem_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	expectedOutput := &dynamodb.PutItemOutput{}

	// Set expectation
	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		// Verify the input
		if aws.ToString(params.TableName) != "test-table" {
			t.Errorf("expected table name test-table, got %s", aws.ToString(params.TableName))
		}

		return expectedOutput, nil
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("test-table"),
		Item: map[string]types.AttributeValue{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
	}

	output, err := mock.PutItem(ctx, input)
	if err != nil {
		t.Fatalf("PutItem failed: %v", err)
	}

	if output != expectedOutput {
		t.Error("PutItem returned unexpected output")
	}
}

func TestMockClient_PutItem_WithError(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	expectedError := errors.New("test error")

	// Set expectation to return error
	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		return nil, expectedError
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("test-table"),
		Item: map[string]types.AttributeValue{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
	}

	_, err := mock.PutItem(ctx, input)
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

func TestMockClient_GetItem_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	expectedItem := map[string]types.AttributeValue{
		"hk": &types.AttributeValueMemberS{Value: "product#P1"},
		"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "P1"},
		}},
	}

	// Set expectation
	mock.GetFunc = func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
		// Verify the input
		if aws.ToString(params.TableName) != "test-table" {
			t.Errorf("expected table name test-table, got %s", aws.ToString(params.TableName))
		}

		return &dynamodb.GetItemOutput{Item: expectedItem}, nil
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String("test-table"),
		Key: map[string]types.AttributeValue{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
	}

	output, err := mock.GetItem(ctx, input)
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if output.Item == nil {
		t.Error("GetItem returned nil item")
	}

	// Verify the returned item
	if hk, exists := output.Item["hk"]; !exists {
		t.Error("item missing hk attribute")
	} else if hkMember, ok := hk.(*types.AttributeValueMemberS); !ok {
		t.Error("hk attribute is not a string")
	} else if hkMember.Value != "product#P1" {
		t.Errorf("expected hk value product#P1, got %s", hkMember.Value)
	}
}

func TestMockClient_Query_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	expectedItems := []map[string]types.AttributeValue{
		{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
		{
			"hk": &types.AttributeValueMemberS{Value: "product#P2"},
			"sk": &types.AttributeValueMemberS{Value: "product#P2"},
		},
	}

	// Set expectation
	mock.QueryFunc = func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
		return &dynamodb.QueryOutput{
			Items: expectedItems,
			Count: int32(len(expectedItems)),
		}, nil
	}

	input := &dynamodb.QueryInput{
		TableName: aws.String("test-table"),
	}

	output, err := mock.Query(ctx, input)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(output.Items) != len(expectedItems) {
		t.Errorf("expected %d items, got %d", len(expectedItems), len(output.Items))
	}

	if output.Count != int32(len(expectedItems)) {
		t.Errorf("expected count %d, got %d", len(expectedItems), output.Count)
	}
}

func TestMockClient_BatchWriteItem_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	// Set expectation
	mock.BatchWriteItemFunc = func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
		// Verify the input has the expected structure
		if len(params.RequestItems) == 0 {
			t.Error("expected request items, got none")
		}

		return &dynamodb.BatchWriteItemOutput{}, nil
	}

	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			"test-table": {
				{
					PutRequest: &types.PutRequest{
						Item: map[string]types.AttributeValue{
							"hk": &types.AttributeValueMemberS{Value: "product#P1"},
							"sk": &types.AttributeValueMemberS{Value: "product#P1"},
						},
					},
				},
			},
		},
	}

	_, err := mock.BatchWriteItem(ctx, input)
	if err != nil {
		t.Fatalf("BatchWriteItem failed: %v", err)
	}
}

func TestMockClient_DeleteItem_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	// Set expectation
	mock.DeleteFunc = func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
		// Verify the input
		if aws.ToString(params.TableName) != "test-table" {
			t.Errorf("expected table name test-table, got %s", aws.ToString(params.TableName))
		}

		return &dynamodb.DeleteItemOutput{}, nil
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("test-table"),
		Key: map[string]types.AttributeValue{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
	}

	_, err := mock.DeleteItem(ctx, input)
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
}

func TestMockClient_UpdateItem_WithExpectation(t *testing.T) {
	mock := NewMockClient(t)
	ctx := context.Background()

	// Set expectation
	mock.UpdateFunc = func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
		// Verify the input
		if aws.ToString(params.TableName) != "test-table" {
			t.Errorf("expected table name test-table, got %s", aws.ToString(params.TableName))
		}

		return &dynamodb.UpdateItemOutput{}, nil
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("test-table"),
		Key: map[string]types.AttributeValue{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		},
	}

	_, err := mock.UpdateItem(ctx, input)
	if err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}
}

// TestMockClient_DefaultBehavior tests that the default functions fail the test
// when called unexpectedly (we can't actually test this without causing the test to fail)
func TestMockClient_DefaultBehavior_Documentation(t *testing.T) {
	// This test documents the default behavior - it would fail if we actually called
	// an operation without setting an expectation

	mock := NewMockClient(t)

	// The default functions will call t.Fatal("unexpected call") if invoked
	// This is the expected behavior for an expectation-based mock

	// We can't actually test this without causing the test to fail,
	// but we can document that this is the expected behavior
	if mock.PutFunc == nil {
		t.Error("PutFunc should be initialized with default behavior")
	}
}
