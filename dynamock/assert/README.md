# Assert Package

The `assert` package provides fluent assertion utilities for testing DynamoDB operations and dynamap entities. It makes tests more readable and maintainable by providing expressive assertion methods that work directly with `*testing.T`.

## Features

- **Fluent API**: Chain assertions for readable test code
- **Standard `*testing.T`**: Works directly with Go's testing framework
- **Multiple Assertion Types**: Support for items, relationships, entities, and DynamoDB items
- **User-Friendly**: Designed for testing user-defined entities and real-world scenarios

## Quick Start

```go
import (
    "testing"
    "github.com/nisimpson/dynamap/dynamock/assert"
)

func TestMyEntities(t *testing.T) {
    // Test DynamoDB query results
    assert.Items(t, queryResult).
        HasCount(3).
        ContainsEntity("order", "O1").
        ContainsRelationship("order", "O1", "product", "P1")

    // Test marshaled relationships
    relationships, _ := dynamap.MarshalRelationships(myEntity)
    assert.Relationships(t, relationships).
        HasCount(2).
        HasSelfRelationship("order", "O1")

    // Test individual DynamoDB items
    assert.DynamoDBItem(t, item).
        IsEntity().
        HasKey("hk", "order#O1").
        HasDataField("customer_id", "C1")
}
```

## Assertion Types

### Items Assertions

Test collections of DynamoDB items (typically from query results):

```go
assert.Items(t, items).
    HasCount(5).                                    // Exact count
    IsNotEmpty().                                   // Has items
    ContainsEntity("product", "P1").                // Contains specific entity
    ContainsRelationship("order", "O1", "product", "P1"). // Contains relationship
    HasAttribute("label", "product").               // At least one item has attribute
    ContainsEntityWithLabel("product").             // Contains entity with label
    ContainsRelationshipWithLabel("products")       // Contains relationship with label
```

### Relationships Assertions

Test dynamap relationship collections:

```go
assert.Relationships(t, relationships).
    HasCount(3).                                    // Exact count
    HasSelfRelationship("order", "O1").             // Has self-relationship
    HasRelationship("order", "O1", "product", "P1"). // Has specific relationship
    HasLabel("order")                               // At least one has label
```

### Entity Assertions

Test TestEntity instances from the dynamock package:

```go
assert.Entity(t, entity).
    CanMarshal().                                   // Marshals without error
    HasSourceID("E1").                              // Has expected source ID
    HasLabel("entity").                             // Has expected label
    HasRefSortKey("sort-key").                      // Has expected ref sort key
    CanMarshalRelationships().                      // Marshals relationships
    HasRelationshipCount(2)                         // Has expected relationship count
```

### DynamoDB Item Assertions

Test individual DynamoDB items:

```go
assert.DynamoDBItem(t, item).
    IsEntity().                                     // Is an entity (hk == sk)
    IsRelationship().                               // Is a relationship (hk != sk)
    HasKey("hk", "order#O1").                       // Has specific key value
    HasAttribute("label", "order").                 // Has specific attribute
    HasDataField("name", "Product Name")            // Has specific data field
```

## Real-World Example

Here's how you would test your own entities:

```go
// Your domain entity
type Order struct {
    ID         string    `json:"id"`
    CustomerID string    `json:"customer_id"`
    Total      int       `json:"total"`
    Products   []Product `json:"-"`
}

func (o *Order) MarshalSelf(opts *dynamap.MarshalOptions) error {
    opts.SourcePrefix = "order"
    opts.SourceID = o.ID
    opts.TargetPrefix = "order"
    opts.TargetID = o.ID
    opts.Label = "order"
    return nil
}

func (o *Order) MarshalRefs(ctx *dynamap.RelationshipContext) error {
    // Add product relationships
    productPtrs := make([]*Product, len(o.Products))
    for i := range o.Products {
        productPtrs[i] = &o.Products[i]
    }
    ctx.AddMany("products", dynamap.SliceOf(productPtrs...))
    return nil
}

// Your test
func TestOrderMarshaling(t *testing.T) {
    order := &Order{
        ID:         "O1",
        CustomerID: "customer@example.com",
        Total:      1000,
        Products:   []Product{{ID: "P1", Name: "Laptop"}},
    }

    // Test marshaling
    relationships, err := dynamap.MarshalRelationships(order)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    // Assert on relationships
    assert.Relationships(t, relationships).
        HasCount(2).                                    // Order + 1 product
        HasSelfRelationship("order", "O1").             // Order entity
        HasRelationship("order", "O1", "product", "P1"). // Order -> Product
        HasLabel("order").                              // Has order label
        HasLabel("order/O1/products")                   // Has products label

    // Test with DynamoDB (simulated response)
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
    }

    // Assert on DynamoDB items
    assert.Items(t, items).
        HasCount(2).
        ContainsEntity("order", "O1").
        ContainsRelationship("order", "O1", "product", "P1")
}
```

## Integration with dynamock

The assert package works perfectly with other dynamock utilities:

```go
// Use with builders
entity := dynamock.NewEntity(
    dynamock.WithID("E1"),
    dynamock.WithPrefix("entity"),
).Build()

assert.Entity(t, entity).CanMarshal()

// Use with presets
order := presets.NewOrder().WithID("O1").Build()
assert.Entity(t, order).HasSourceID("O1")

// Use with mock clients
mock := dynamock.NewMockClient(t)
// ... set up expectations
assert.Items(t, capturedItems).HasCount(expected)
```
