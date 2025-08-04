package assert_test

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock/assert"
)

// This example shows how a user would define their own entities
// and use the assert package to test them

type UserProduct struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int    `json:"price"`
}

func (p *UserProduct) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "product"
	opts.SourceID = p.ID
	opts.TargetPrefix = "product"
	opts.TargetID = p.ID
	opts.Label = "product"
	opts.RefSortKey = p.Category
	return nil
}

type UserOrder struct {
	ID         string        `json:"id"`
	CustomerID string        `json:"customer_id"`
	Total      int           `json:"total"`
	CreatedAt  time.Time     `json:"created_at"`
	Products   []UserProduct `json:"-"`
}

func (o *UserOrder) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "order"
	opts.SourceID = o.ID
	opts.TargetPrefix = "order"
	opts.TargetID = o.ID
	opts.Label = "order"
	opts.RefSortKey = o.CreatedAt.Format(time.RFC3339)
	opts.Created = o.CreatedAt
	return nil
}

func (o *UserOrder) MarshalRefs(ctx *dynamap.RelationshipContext) error {
	// Convert products to marshalers
	productPtrs := make([]*UserProduct, len(o.Products))
	for i := range o.Products {
		productPtrs[i] = &o.Products[i]
	}
	ctx.AddMany("products", dynamap.SliceOf(productPtrs...))
	return nil
}

// Example_userTestingWorkflow demonstrates how a user would test their entities
func Example_userTestingWorkflow() {
	// This would be a real test function in user code
	t := &testing.T{} // In real usage, this comes from the test function parameter

	// User creates their domain entities
	laptop := &UserProduct{
		ID:       "P1",
		Name:     "Gaming Laptop",
		Category: "electronics",
		Price:    1299,
	}

	mouse := &UserProduct{
		ID:       "P2",
		Name:     "Wireless Mouse",
		Category: "electronics",
		Price:    49,
	}

	order := &UserOrder{
		ID:         "O1",
		CustomerID: "customer@example.com",
		Total:      1348,
		CreatedAt:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Products:   []UserProduct{*laptop, *mouse},
	}

	// User marshals their order to relationships
	relationships, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("Failed to marshal order: %v", err)
	}

	// User tests the marshaled relationships
	assert.Relationships(t, relationships).
		HasCount(3).                                     // 1 order + 2 products
		HasSelfRelationship("order", "O1").              // Order entity exists
		HasRelationship("order", "O1", "product", "P1"). // Order -> Laptop
		HasRelationship("order", "O1", "product", "P2"). // Order -> Mouse
		HasLabel("order").                               // Has order label
		HasLabel("order/O1/products")                    // Has products relationship label

	// User simulates what DynamoDB would return (in real tests, this comes from actual DynamoDB queries)
	items := []map[string]types.AttributeValue{
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
		},
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
		},
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P2"},
			"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
		},
	}

	// User tests the DynamoDB query results
	assert.Items(t, items).
		HasCount(3).
		IsNotEmpty().
		ContainsEntity("order", "O1").
		ContainsRelationship("order", "O1", "product", "P1").
		ContainsRelationship("order", "O1", "product", "P2").
		HasAttribute("label", "order").
		ContainsEntityWithLabel("order").
		ContainsRelationshipWithLabel("products")

	// User tests individual DynamoDB items
	for _, item := range items {
		assert.DynamoDBItem(t, item).
			HasKey("hk", "order#O1") // All items in this example have the same source

		// Check if it's an entity or relationship
		if hk, exists := item["hk"]; exists {
			if sk, exists := item["sk"]; exists {
				if hkStr, ok := hk.(*types.AttributeValueMemberS); ok {
					if skStr, ok := sk.(*types.AttributeValueMemberS); ok {
						if hkStr.Value == skStr.Value {
							assert.DynamoDBItem(t, item).IsEntity()
						} else {
							assert.DynamoDBItem(t, item).IsRelationship()
						}
					}
				}
			}
		}
	}
}

// TestRealUserScenario demonstrates a complete test that a user might write
func TestRealUserScenario(t *testing.T) {
	// User creates test data
	product := &UserProduct{
		ID:       "LAPTOP001",
		Name:     "MacBook Pro",
		Category: "computers",
		Price:    2499,
	}

	order := &UserOrder{
		ID:         "ORDER123",
		CustomerID: "john.doe@example.com",
		Total:      2499,
		CreatedAt:  time.Date(2025, 2, 1, 14, 30, 0, 0, time.UTC),
		Products:   []UserProduct{*product},
	}

	// User marshals and tests relationships
	relationships, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("Failed to marshal order: %v", err)
	}

	// User verifies the relationships are correct
	assert.Relationships(t, relationships).
		HasCount(2).                                                  // 1 order + 1 product
		HasSelfRelationship("order", "ORDER123").                     // Order exists
		HasRelationship("order", "ORDER123", "product", "LAPTOP001"). // Order -> Product
		HasLabel("order").                                            // Has order label
		HasLabel("order/ORDER123/products")                           // Has products label

	// User would typically store these in DynamoDB and then query them back
	// For this test, we simulate the DynamoDB response
	queryResult := []map[string]types.AttributeValue{
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#ORDER123"},
			"sk":    &types.AttributeValueMemberS{Value: "order#ORDER123"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":          &types.AttributeValueMemberS{Value: "ORDER123"},
				"customer_id": &types.AttributeValueMemberS{Value: "john.doe@example.com"},
				"total":       &types.AttributeValueMemberN{Value: "2499"},
			}},
		},
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#ORDER123"},
			"sk":    &types.AttributeValueMemberS{Value: "product#LAPTOP001"},
			"label": &types.AttributeValueMemberS{Value: "order/ORDER123/products"},
			"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":       &types.AttributeValueMemberS{Value: "LAPTOP001"},
				"name":     &types.AttributeValueMemberS{Value: "MacBook Pro"},
				"category": &types.AttributeValueMemberS{Value: "computers"},
				"price":    &types.AttributeValueMemberN{Value: "2499"},
			}},
		},
	}

	// User tests the query results
	assert.Items(t, queryResult).
		HasCount(2).
		ContainsEntity("order", "ORDER123").
		ContainsRelationship("order", "ORDER123", "product", "LAPTOP001").
		HasAttribute("label", "order")

	// User tests specific items
	orderItem := queryResult[0]
	assert.DynamoDBItem(t, orderItem).
		IsEntity().
		HasKey("hk", "order#ORDER123").
		HasKey("sk", "order#ORDER123").
		HasAttribute("label", "order").
		HasDataField("customer_id", "john.doe@example.com")

	relationshipItem := queryResult[1]
	assert.DynamoDBItem(t, relationshipItem).
		IsRelationship().
		HasKey("hk", "order#ORDER123").
		HasKey("sk", "product#LAPTOP001").
		HasAttribute("label", "order/ORDER123/products").
		HasDataField("name", "MacBook Pro").
		HasDataField("category", "computers")
}
