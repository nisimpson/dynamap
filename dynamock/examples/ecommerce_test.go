package examples

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

func TestNewProduct(t *testing.T) {
	product := NewProduct().
		WithID("P1").
		WithCategory("electronics").
		WithPrice(299).
		WithName("Laptop").
		Build()

	// Test that it implements all dynamap interfaces
	var _ dynamap.Marshaler = product
	var _ dynamap.RefMarshaler = product
	var _ dynamap.Unmarshaler = product
	var _ dynamap.RefUnmarshaler = product

	// Test marshaling
	opts := &dynamap.MarshalOptions{}
	err := product.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "P1" {
		t.Errorf("expected source ID 'P1', got %s", opts.SourceID)
	}

	if opts.Label != "product" {
		t.Errorf("expected label 'product', got %s", opts.Label)
	}

	if opts.RefSortKey != "electronics" {
		t.Errorf("expected ref sort key 'electronics', got %s", opts.RefSortKey)
	}
}

func TestNewOrder(t *testing.T) {
	product1 := QuickProduct("P1", "electronics")
	product2 := QuickProduct("P2", "books")

	order := NewOrder().
		WithID("O1").
		WithCustomerID("C1").
		WithProduct(product1).
		WithProducts(product2).
		Build()

	// Test that it implements all dynamap interfaces
	var _ dynamap.Marshaler = order
	var _ dynamap.RefMarshaler = order
	var _ dynamap.Unmarshaler = order
	var _ dynamap.RefUnmarshaler = order

	// Test marshaling
	opts := &dynamap.MarshalOptions{}
	err := order.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "O1" {
		t.Errorf("expected source ID 'O1', got %s", opts.SourceID)
	}

	if opts.Label != "order" {
		t.Errorf("expected label 'order', got %s", opts.Label)
	}

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

func TestNewCustomer(t *testing.T) {
	customer := NewCustomer().
		WithID("C1").
		WithEmail("test@example.com").
		WithName("John Doe").
		WithTier("premium").
		Build()

	// Test that it implements all dynamap interfaces
	var _ dynamap.Marshaler = customer
	var _ dynamap.RefMarshaler = customer
	var _ dynamap.Unmarshaler = customer
	var _ dynamap.RefUnmarshaler = customer

	// Test marshaling
	opts := &dynamap.MarshalOptions{}
	err := customer.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "C1" {
		t.Errorf("expected source ID 'C1', got %s", opts.SourceID)
	}

	if opts.Label != "customer" {
		t.Errorf("expected label 'customer', got %s", opts.Label)
	}

	if opts.RefSortKey != "premium" {
		t.Errorf("expected ref sort key 'premium', got %s", opts.RefSortKey)
	}
}

func TestCustomerBuilder_TierMethods(t *testing.T) {
	premiumCustomer := NewCustomer().
		WithID("C1").
		WithEmail("premium@example.com").
		Premium().
		Build()

	opts := &dynamap.MarshalOptions{}
	err := premiumCustomer.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.RefSortKey != "premium" {
		t.Errorf("expected premium tier, got %s", opts.RefSortKey)
	}

	standardCustomer := NewCustomer().
		WithID("C2").
		WithEmail("standard@example.com").
		Standard().
		Build()

	opts = &dynamap.MarshalOptions{}
	err = standardCustomer.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.RefSortKey != "standard" {
		t.Errorf("expected standard tier, got %s", opts.RefSortKey)
	}
}

func TestQuickFunctions(t *testing.T) {
	// Test QuickProduct
	product := QuickProduct("P1", "electronics")
	opts := &dynamap.MarshalOptions{}
	err := product.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "P1" {
		t.Errorf("expected product ID 'P1', got %s", opts.SourceID)
	}

	if opts.RefSortKey != "electronics" {
		t.Errorf("expected product category 'electronics', got %s", opts.RefSortKey)
	}

	// Test QuickOrder
	order := QuickOrder("O1", "C1")
	opts = &dynamap.MarshalOptions{}
	err = order.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "O1" {
		t.Errorf("expected order ID 'O1', got %s", opts.SourceID)
	}

	// Test QuickCustomer
	customer := QuickCustomer("C1", "test@example.com")
	opts = &dynamap.MarshalOptions{}
	err = customer.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "C1" {
		t.Errorf("expected customer ID 'C1', got %s", opts.SourceID)
	}

	if opts.RefSortKey != "standard" {
		t.Errorf("expected standard tier, got %s", opts.RefSortKey)
	}

	// Test QuickPremiumCustomer
	premiumCustomer := QuickPremiumCustomer("C2", "premium@example.com")
	opts = &dynamap.MarshalOptions{}
	err = premiumCustomer.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "C2" {
		t.Errorf("expected customer ID 'C2', got %s", opts.SourceID)
	}

	if opts.RefSortKey != "premium" {
		t.Errorf("expected premium tier, got %s", opts.RefSortKey)
	}
}

func TestPresets_WithDynamapIntegration(t *testing.T) {
	// Test that preset entities work with dynamap operations
	product := NewProduct().
		WithID("P1").
		WithCategory("electronics").
		WithPrice(299).
		Build()

	// Test MarshalSelf
	opts := &dynamap.MarshalOptions{}
	err := product.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "P1" {
		t.Errorf("expected source ID 'P1', got %s", opts.SourceID)
	}

	if opts.SourcePrefix != "product" {
		t.Errorf("expected source prefix 'product', got %s", opts.SourcePrefix)
	}

	if opts.Label != "product" {
		t.Errorf("expected label 'product', got %s", opts.Label)
	}

	if opts.RefSortKey != "electronics" {
		t.Errorf("expected ref sort key 'electronics', got %s", opts.RefSortKey)
	}
}

func TestPresets_WithMockClient(t *testing.T) {
	// Test that preset entities work with the mock client
	mock := dynamock.NewMockClient(t)
	table := dynamap.NewTable("test-table")

	product := NewProduct().
		WithID("P1").
		WithCategory("electronics").
		WithPrice(299).
		WithName("Gaming Laptop").
		Build()

	// Set expectation
	mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
		// Just verify it doesn't crash
		return &dynamodb.PutItemOutput{}, nil
	}

	// Test marshaling and putting
	putInput, err := table.MarshalPut(product)
	if err != nil {
		t.Fatalf("MarshalPut failed: %v", err)
	}

	_, err = mock.PutItem(context.Background(), putInput)
	if err != nil {
		t.Fatalf("PutItem failed: %v", err)
	}
}

func TestPresets_ComplexOrder(t *testing.T) {
	// Test creating a complex order with multiple products
	products := []*dynamock.TestEntity{
		NewProduct().WithID("P1").WithCategory("electronics").WithPrice(299).Build(),
		NewProduct().WithID("P2").WithCategory("books").WithPrice(19).Build(),
		NewProduct().WithID("P3").WithCategory("electronics").WithPrice(599).Build(),
	}

	// Convert to Marshaler slice
	productMarshalers := make([]dynamap.Marshaler, len(products))
	for i, product := range products {
		productMarshalers[i] = product
	}

	order := NewOrder().
		WithID("O1").
		WithCustomerID("C1").
		WithProducts(productMarshalers...).
		Build()

	// Test that it can marshal relationships
	relationshipList, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("MarshalRelationships failed: %v", err)
	}

	// Should have 4 relationships: 1 self + 3 products
	if len(relationshipList) != 4 {
		t.Errorf("expected 4 relationships, got %d", len(relationshipList))
	}
}
