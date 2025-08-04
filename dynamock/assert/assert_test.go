package assert

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

// User-defined entities for testing - these simulate what users would create
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
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
	Total      int       `json:"total"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Products   []Product `json:"-"`
}

func (o *Order) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = "order"
	opts.SourceID = o.ID
	opts.TargetPrefix = "order"
	opts.TargetID = o.ID
	opts.Label = "order"
	opts.RefSortKey = o.CreatedAt.Format(time.RFC3339)
	opts.Created = o.CreatedAt
	return nil
}

func (o *Order) MarshalRefs(ctx *dynamap.RelationshipContext) error {
	// Convert products to marshalers
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
		o.CreatedAt = rel.CreatedAt
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

// TestItemsAssertion demonstrates how users would test DynamoDB query results
func TestItemsAssertion(t *testing.T) {
	// Create test data that simulates what DynamoDB would return
	items := []map[string]types.AttributeValue{
		{
			"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "product"},
			"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":       &types.AttributeValueMemberS{Value: "P1"},
				"name":     &types.AttributeValueMemberS{Value: "Laptop"},
				"category": &types.AttributeValueMemberS{Value: "electronics"},
				"price":    &types.AttributeValueMemberN{Value: "999"},
			}},
		},
		{
			"hk":    &types.AttributeValueMemberS{Value: "product#P2"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P2"},
			"label": &types.AttributeValueMemberS{Value: "product"},
			"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":       &types.AttributeValueMemberS{Value: "P2"},
				"name":     &types.AttributeValueMemberS{Value: "Book"},
				"category": &types.AttributeValueMemberS{Value: "books"},
				"price":    &types.AttributeValueMemberN{Value: "25"},
			}},
		},
		{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
			"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"id":       &types.AttributeValueMemberS{Value: "P1"},
				"name":     &types.AttributeValueMemberS{Value: "Laptop"},
				"category": &types.AttributeValueMemberS{Value: "electronics"},
				"price":    &types.AttributeValueMemberN{Value: "999"},
			}},
		},
	}

	// Test basic count assertions
	Items(t, items).HasCount(3)
	Items(t, items).IsNotEmpty()
	Items(t, []map[string]types.AttributeValue{}).IsEmpty()

	// Test entity detection
	Items(t, items).ContainsEntity("product", "P1")
	Items(t, items).ContainsEntity("product", "P2")

	// Test relationship detection
	Items(t, items).ContainsRelationship("order", "O1", "product", "P1")

	// Test attribute assertions
	Items(t, items).HasAttribute("label", "product")
	Items(t, items).ContainsEntityWithLabel("product")
	Items(t, items).ContainsRelationshipWithLabel("products")
}

// TestRelationshipsAssertion demonstrates testing marshaled relationships
func TestRelationshipsAssertion(t *testing.T) {
	// Create a test order with products
	order := &Order{
		ID:         "O1",
		CustomerID: "C1",
		Total:      1024,
		Status:     "pending",
		CreatedAt:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Products: []Product{
			{ID: "P1", Name: "Laptop", Category: "electronics", Price: 999},
			{ID: "P2", Name: "Book", Category: "books", Price: 25},
		},
	}

	// Marshal the relationships
	relationships, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("Failed to marshal relationships: %v", err)
	}

	// Convert to the format expected by our assertion
	relSlice := make([]dynamap.Relationship, len(relationships))
	for i, rel := range relationships {
		relSlice[i] = rel
	}

	// Test relationship assertions
	Relationships(t, relSlice).HasCount(3) // order + 2 products
	Relationships(t, relSlice).HasSelfRelationship("order", "O1")
	Relationships(t, relSlice).HasRelationship("order", "O1", "product", "P1")
	Relationships(t, relSlice).HasRelationship("order", "O1", "product", "P2")
	Relationships(t, relSlice).HasLabel("order")
	Relationships(t, relSlice).HasLabel("order/O1/products")
}

// TestEntityAssertion demonstrates testing entity marshaling behavior
func TestEntityAssertion(t *testing.T) {
	// Create a test entity using the dynamock TestEntity
	entity := dynamock.NewEntity(
		dynamock.WithID("E1"),
		dynamock.WithPrefix("test"),
		dynamock.WithLabel("test-entity"),
		dynamock.WithData(map[string]interface{}{
			"name":        "Test Entity",
			"description": "A test entity for assertions",
		}),
	).Build()

	// Test entity marshaling assertions
	Entity(t, entity).CanMarshal()
	Entity(t, entity).HasSourceID("E1")
	Entity(t, entity).HasLabel("test-entity")

	// Test entity with relationships
	child1 := dynamock.NewEntity(
		dynamock.WithID("C1"),
		dynamock.WithPrefix("child"),
		dynamock.WithLabel("child-entity"),
	).Build()

	child2 := dynamock.NewEntity(
		dynamock.WithID("C2"),
		dynamock.WithPrefix("child"),
		dynamock.WithLabel("child-entity"),
	).Build()

	entityWithRels := dynamock.NewEntity(
		dynamock.WithID("E2"),
		dynamock.WithPrefix("parent"),
		dynamock.WithLabel("parent-entity"),
		dynamock.WithRelationships("children", child1, child2),
	).Build()

	Entity(t, entityWithRels).CanMarshalRelationships()
	Entity(t, entityWithRels).HasRelationshipCount(3) // parent + 2 children
}

// TestDynamoDBItemAssertion demonstrates testing individual DynamoDB items
func TestDynamoDBItemAssertion(t *testing.T) {
	// Create a test item that represents a product entity
	productItem := map[string]types.AttributeValue{
		"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
		"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
		"label": &types.AttributeValueMemberS{Value: "product"},
		"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberS{Value: "P1"},
			"name":     &types.AttributeValueMemberS{Value: "Laptop"},
			"category": &types.AttributeValueMemberS{Value: "electronics"},
		}},
		"created_at": &types.AttributeValueMemberS{Value: "2025-01-01T12:00:00Z"},
	}

	// Test item structure assertions
	DynamoDBItem(t, productItem).IsEntity()
	DynamoDBItem(t, productItem).HasKey("hk", "product#P1")
	DynamoDBItem(t, productItem).HasKey("sk", "product#P1")
	DynamoDBItem(t, productItem).HasAttribute("label", "product")
	DynamoDBItem(t, productItem).HasDataField("name", "Laptop")
	DynamoDBItem(t, productItem).HasDataField("category", "electronics")

	// Create a test item that represents a relationship
	relationshipItem := map[string]types.AttributeValue{
		"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
		"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
		"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
		"data": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"id":   &types.AttributeValueMemberS{Value: "P1"},
			"name": &types.AttributeValueMemberS{Value: "Laptop"},
		}},
	}

	DynamoDBItem(t, relationshipItem).IsRelationship()
	DynamoDBItem(t, relationshipItem).HasKey("hk", "order#O1")
	DynamoDBItem(t, relationshipItem).HasKey("sk", "product#P1")
	DynamoDBItem(t, relationshipItem).HasAttribute("label", "order/O1/products")
}

// TestUserWorkflow demonstrates a complete user testing workflow
func TestUserWorkflow(t *testing.T) {
	// Step 1: User creates their domain entities
	product1 := &Product{ID: "P1", Name: "Laptop", Category: "electronics", Price: 999}
	product2 := &Product{ID: "P2", Name: "Book", Category: "books", Price: 25}

	order := &Order{
		ID:         "O1",
		CustomerID: "C1",
		Total:      1024,
		Status:     "pending",
		CreatedAt:  time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Products:   []Product{*product1, *product2},
	}

	// Step 2: User marshals their entities for storage
	relationships, err := dynamap.MarshalRelationships(order)
	if err != nil {
		t.Fatalf("Failed to marshal order: %v", err)
	}

	// Step 3: User converts relationships to DynamoDB items (simulated)
	items := make([]map[string]types.AttributeValue, len(relationships))
	for i, rel := range relationships {
		// This would normally be done by the DynamoDB client
		items[i] = map[string]types.AttributeValue{
			"hk":    &types.AttributeValueMemberS{Value: rel.Source},
			"sk":    &types.AttributeValueMemberS{Value: rel.Target},
			"label": &types.AttributeValueMemberS{Value: rel.Label},
			"data":  &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{}}, // Simplified
		}
	}

	// Step 4: User tests their results using the assert package
	Items(t, items).
		HasCount(3).                                          // Order + 2 products
		ContainsEntity("order", "O1").                        // Order entity exists
		ContainsRelationship("order", "O1", "product", "P1"). // Order -> Product 1
		ContainsRelationship("order", "O1", "product", "P2")  // Order -> Product 2

	// Note: We can't test ContainsEntity for products in this simplified example
	// because the marshaled relationships don't include the individual product entities,
	// only the relationships from order to products.

	// Step 5: User tests individual items
	for _, item := range items {
		if hk, exists := item["hk"]; exists {
			if hkStr, ok := hk.(*types.AttributeValueMemberS); ok {
				if hkStr.Value == "order#O1" {
					if sk, exists := item["sk"]; exists {
						if skStr, ok := sk.(*types.AttributeValueMemberS); ok {
							if skStr.Value == "order#O1" {
								DynamoDBItem(t, item).IsEntity()
							} else {
								DynamoDBItem(t, item).IsRelationship()
							}
						}
					}
				}
			}
		}
	}
}

// TestAssertionFailures demonstrates that assertions properly fail when conditions aren't met
// This test is commented out for now to focus on the main functionality
/*
func TestAssertionFailures(t *testing.T) {
	// This would test assertion failures but requires more complex mocking
	// For now, we focus on demonstrating successful assertion usage
}
*/
