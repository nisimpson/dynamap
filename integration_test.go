package dynamap

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// TestQueryingEntities demonstrates querying for entities
func TestQueryingEntities(t *testing.T) {
	t.Skip("Skipping AWS integration test")

	ctx := context.Background()
	cfg, _ := config.LoadDefaultConfig(ctx)
	ddb := dynamodb.NewFromConfig(cfg)
	table := NewTable("ecommerce-table")

	// Query all products in electronics category
	queryList := &QueryList{
		Label:          "product",
		RefSortFilter:  expression.Key(AttributeNameRefSortKey).BeginsWith("electronics"),
		Limit:          10,
		SortDescending: false,
	}

	queryInput, err := table.MarshalQuery(queryList)
	if err != nil {
		log.Fatal(err)
	}

	result, err := ddb.Query(ctx, queryInput)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the results
	products := make([]Product, 0, len(result.Items))

	relationships, err := UnmarshalList(result.Items, &products)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d products in electronics category\n", len(relationships))

	// Output:
	// Found 0 products in electronics category
}

// ExampleQueryingEntityRelationships demonstrates querying an entity and its relationships
func TestQueryingEntityRelationships(t *testing.T) {
	t.Skip("Skipping AWS integration test")

	ctx := context.Background()
	cfg, _ := config.LoadDefaultConfig(ctx)
	ddb := dynamodb.NewFromConfig(cfg)
	table := NewTable("ecommerce-table")

	// Create an order to query
	order := &Order{ID: "O001"}

	// Query the order and all its relationships
	queryEntity := &QueryEntity{
		Source:         order,
		Limit:          50,
		SortDescending: false,
	}

	queryInput, err := table.MarshalQuery(queryEntity)
	if err != nil {
		log.Fatal(err)
	}

	result, err := ddb.Query(ctx, queryInput)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the entity and its relationships
	var fullOrder Order
	relationships, err := UnmarshalEntity(result.Items, &fullOrder)
	if err != nil {
		if err == ErrItemNotFound {
			fmt.Println("Order not found")
			return
		}
		log.Fatal(err)
	}

	fmt.Printf("Order %s has %d relationships\n", fullOrder.ID, len(relationships)-1) // -1 for self
	fmt.Printf("Order contains %d products\n", len(fullOrder.Products))

	// Output:
	// Order not found
}

// ExampleAdvancedQuerying demonstrates advanced query filtering
func TestAdvancedQuerying(t *testing.T) {
	t.Skip("Skipping AWS integration test")

	ctx := context.Background()
	cfg, _ := config.LoadDefaultConfig(ctx)
	ddb := dynamodb.NewFromConfig(cfg)
	table := NewTable("ecommerce-table")

	// Query products with advanced filtering
	queryList := &QueryList{
		Label: "product",
		// Filter by category prefix
		RefSortFilter: expression.Key("gsi1_sk").BeginsWith("electronics"),
		// Additional filter on the data
		ConditionFilter: expression.Name("data.category").Equal(expression.Value("electronics")),
		Limit:           20,
		SortDescending:  true, // Newest first
	}

	queryInput, err := table.MarshalQuery(queryList)
	if err != nil {
		log.Fatal(err)
	}

	result, err := ddb.Query(ctx, queryInput)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d affordable electronics products\n", len(result.Items))

	// Output:
	// Found 0 affordable electronics products
}
