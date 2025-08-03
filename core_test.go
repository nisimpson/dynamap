package dynamap

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Test entities for core functionality
type Order struct {
	ID          string    `dynamodbav:"id"`
	PurchasedBy string    `dynamodbav:"purchased_by"`
	Products    []Product `dynamodbav:"-"`
	Created     time.Time `dynamodbav:"-"`
	Updated     time.Time `dynamodbav:"-"`
}

func (o *Order) MarshalSelf(opts *MarshalOptions) error {
	opts.WithSelfTarget("order", o.ID)
	opts.RefSortKey = opts.Created.Format(time.RFC3339)
	opts.Created = o.Created
	opts.Updated = o.Updated
	return nil
}

func (o *Order) MarshalRefs(ctx *RelationshipContext) error {
	productPtrs := make([]*Product, len(o.Products))
	for i := range o.Products {
		productPtrs[i] = &o.Products[i]
	}
	ctx.AddMany("products", SliceOf(productPtrs...))
	return nil
}

func (o *Order) UnmarshalSelf(rel *Relationship) error {
	o.Created = rel.CreatedAt
	o.Updated = rel.UpdatedAt
	return nil
}

func (o *Order) UnmarshalRef(name string, id string, ref *Relationship) error {
	if name == "products" {
		var product Product
		product.ID = id
		o.Products = append(o.Products, product)
	}
	return nil
}

type Product struct {
	ID       string `dynamodbav:"id"`
	Category string `dynamodbav:"category"`
}

func (p *Product) MarshalSelf(opts *MarshalOptions) error {
	opts.SourcePrefix = "product"
	opts.SourceID = p.ID
	opts.TargetPrefix = "product"
	opts.TargetID = p.ID
	opts.Label = "product"
	opts.RefSortKey = p.Category
	return nil
}

// Tests for core functionality

func TestNewTable(t *testing.T) {
	table := NewTable("test-table")

	if table.TableName != "test-table" {
		t.Errorf("Expected table name 'test-table', got %s", table.TableName)
	}
	if table.RefIndexName != "ref-index" {
		t.Errorf("Expected ref index name 'ref-index', got %s", table.RefIndexName)
	}
	if table.KeyDelimiter != "#" {
		t.Errorf("Expected key delimiter '#', got %s", table.KeyDelimiter)
	}
	if table.PaginationTTL != 24*time.Hour {
		t.Errorf("Expected pagination TTL 24h, got %v", table.PaginationTTL)
	}
}

func TestDefaultClock(t *testing.T) {
	now := DefaultClock()
	if now.IsZero() {
		t.Error("DefaultClock returned zero time")
	}
}

func TestSliceOf(t *testing.T) {
	products := []*Product{
		{ID: "P1", Category: "electronics"},
		{ID: "P2", Category: "books"},
	}

	marshalers := SliceOf(products[0], products[1])

	if len(marshalers) != 2 {
		t.Errorf("Expected 2 marshalers, got %d", len(marshalers))
	}

	for i, m := range marshalers {
		if _, ok := m.(Marshaler); !ok {
			t.Errorf("Item %d does not implement Marshaler", i)
		}
	}
}

func TestMarshalRelationships(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("single entity", func(t *testing.T) {
		product := &Product{ID: "P1", Category: "electronics"}

		relationships, err := MarshalRelationships(product, func(opts *MarshalOptions) {
			opts.Tick = func() time.Time { return fixedTime }
			opts.Created = fixedTime
			opts.Updated = fixedTime
		})

		if err != nil {
			t.Fatalf("Failed to marshal relationships: %v", err)
		}

		if len(relationships) != 1 {
			t.Errorf("Expected 1 relationship, got %d", len(relationships))
		}

		rel := relationships[0]
		if rel.Source != "product#P1" {
			t.Errorf("Expected source 'product#P1', got %s", rel.Source)
		}
		if rel.Target != "product#P1" {
			t.Errorf("Expected target 'product#P1', got %s", rel.Target)
		}
		if rel.Label != "product" {
			t.Errorf("Expected label 'product', got %s", rel.Label)
		}
	})

	t.Run("entity with relationships", func(t *testing.T) {
		order := &Order{
			ID:          "O1",
			PurchasedBy: "john",
			Products: []Product{
				{ID: "P1", Category: "electronics"},
				{ID: "P2", Category: "books"},
			},
		}

		relationships, err := MarshalRelationships(order, func(opts *MarshalOptions) {
			opts.Tick = func() time.Time { return fixedTime }
			opts.Created = fixedTime
			opts.Updated = fixedTime
		})

		if err != nil {
			t.Fatalf("Failed to marshal relationships: %v", err)
		}

		if len(relationships) != 3 {
			t.Errorf("Expected 3 relationships, got %d", len(relationships))
		}

		// Check self relationship
		selfRel := relationships[0]
		if selfRel.Source != "order#O1" {
			t.Errorf("Expected source 'order#O1', got %s", selfRel.Source)
		}
		if selfRel.Label != "order" {
			t.Errorf("Expected label 'order', got %s", selfRel.Label)
		}

		// Check product relationships
		for i := 1; i < len(relationships); i++ {
			rel := relationships[i]
			if rel.Source != "order#O1" {
				t.Errorf("Expected source 'order#O1', got %s", rel.Source)
			}
			if rel.Label != "order/O1/products" {
				t.Errorf("Expected label 'order/O1/products', got %s", rel.Label)
			}
		}
	})

	t.Run("skip relationships", func(t *testing.T) {
		order := &Order{
			ID:          "O1",
			PurchasedBy: "john",
			Products: []Product{
				{ID: "P1", Category: "electronics"},
			},
		}

		relationships, err := MarshalRelationships(order, func(opts *MarshalOptions) {
			opts.SkipRefs = true
		})

		if err != nil {
			t.Fatalf("Failed to marshal relationships: %v", err)
		}

		if len(relationships) != 1 {
			t.Errorf("Expected 1 relationship when skipping refs, got %d", len(relationships))
		}
	})

	t.Run("with preset timestamps", func(t *testing.T) {
		product := &Product{ID: "P1", Category: "electronics"}

		relationships, err := MarshalRelationships(product, func(opts *MarshalOptions) {
			opts.Created = fixedTime
			opts.Updated = fixedTime.Add(time.Hour)
		})

		if err != nil {
			t.Fatalf("Failed to marshal relationships: %v", err)
		}

		rel := relationships[0]
		if !rel.CreatedAt.Equal(fixedTime) {
			t.Errorf("Expected CreatedAt to be %v, got %v", fixedTime, rel.CreatedAt)
		}
		if !rel.UpdatedAt.Equal(fixedTime.Add(time.Hour)) {
			t.Errorf("Expected UpdatedAt to be %v, got %v", fixedTime.Add(time.Hour), rel.UpdatedAt)
		}
	})
}

func TestMarshalOptionsHelpers(t *testing.T) {
	opts := MarshalOptions{
		SourcePrefix:   "order",
		SourceID:       "O1",
		TargetPrefix:   "product",
		TargetID:       "P1",
		KeyDelimiter:   "#",
		LabelDelimiter: "/",
	}

	t.Run("sourceKey", func(t *testing.T) {
		expected := "order#O1"
		if got := opts.sourceKey(); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})

	t.Run("targetKey", func(t *testing.T) {
		expected := "product#P1"
		if got := opts.targetKey(); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})

	t.Run("refLabel", func(t *testing.T) {
		expected := "order/O1/products"
		if got := opts.refLabel("products"); got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	})

	t.Run("splitLabel", func(t *testing.T) {
		t.Run("single part", func(t *testing.T) {
			rel := Relationship{Label: "order"}
			prefix, id, name, err := opts.splitLabel(rel)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if prefix != "order" || id != "" || name != "" {
				t.Errorf("Expected ('order', '', ''), got ('%s', '%s', '%s')", prefix, id, name)
			}
		})

		t.Run("three parts", func(t *testing.T) {
			rel := Relationship{Label: "order/O1/products"}
			prefix, id, name, err := opts.splitLabel(rel)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if prefix != "order" || id != "O1" || name != "products" {
				t.Errorf("Expected ('order', 'O1', 'products'), got ('%s', '%s', '%s')", prefix, id, name)
			}
		})
	})
}

func TestRelationshipContext(t *testing.T) {
	ctx := &RelationshipContext{
		source: "order#O1",
		opts: MarshalOptions{
			SourcePrefix:   "order",
			SourceID:       "O1",
			KeyDelimiter:   "#",
			LabelDelimiter: "/",
			Tick:           DefaultClock,
		},
	}

	t.Run("AddOne", func(t *testing.T) {
		product := &Product{ID: "P1", Category: "electronics"}
		ctx.AddOne("products", product)

		if ctx.err != nil {
			t.Fatalf("Unexpected error: %v", ctx.err)
		}

		if len(ctx.refs) != 1 {
			t.Errorf("Expected 1 reference, got %d", len(ctx.refs))
		}

		ref := ctx.refs[0]
		if ref.Label != "order/O1/products" {
			t.Errorf("Expected label 'order/O1/products', got %s", ref.Label)
		}
	})

	t.Run("AddMany", func(t *testing.T) {
		ctx.refs = nil // Reset
		products := []*Product{
			{ID: "P1", Category: "electronics"},
			{ID: "P2", Category: "books"},
		}

		ctx.AddMany("products", SliceOf(products[0], products[1]))

		if ctx.err != nil {
			t.Fatalf("Unexpected error: %v", ctx.err)
		}

		if len(ctx.refs) != 2 {
			t.Errorf("Expected 2 references, got %d", len(ctx.refs))
		}
	})
}

func TestUnmarshalSelf(t *testing.T) {
	t.Run("valid item", func(t *testing.T) {
		// Create proper entity data
		productData := &Product{ID: "P1", Category: "electronics"}
		dataAttr, err := attributevalue.Marshal(productData)
		if err != nil {
			t.Fatalf("Failed to marshal product data: %v", err)
		}

		item := Item{
			"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "product"},
			"data":  dataAttr,
		}

		var product Product
		rel, err := UnmarshalSelf(item, &product)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if rel.Source != "product#P1" {
			t.Errorf("Expected source 'product#P1', got %s", rel.Source)
		}
		if rel.Label != "product" {
			t.Errorf("Expected label 'product', got %s", rel.Label)
		}
	})

	t.Run("invalid relationship", func(t *testing.T) {
		invalidItem := Item{
			"invalid": &types.AttributeValueMemberS{Value: "test"},
		}

		var product Product
		_, err := UnmarshalSelf(invalidItem, &product)
		if err == nil {
			t.Error("Expected error from invalid item")
		}
	})

	t.Run("missing data attribute", func(t *testing.T) {
		item := Item{
			"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "product"},
		}

		var product Product
		_, err := UnmarshalSelf(item, &product)
		if err == nil {
			t.Error("Expected error from missing data attribute")
		}
	})

	t.Run("invalid data type", func(t *testing.T) {
		item := Item{
			"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "product"},
			"data":  &types.AttributeValueMemberBOOL{Value: true},
		}

		var product Product
		_, err := UnmarshalSelf(item, &product)
		if err == nil {
			t.Error("Expected error from invalid data type")
		}
	})
}

func TestUnmarshalTableKey(t *testing.T) {
	t.Run("valid item", func(t *testing.T) {
		item := Item{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		}

		source, target, err := UnmarshalTableKey(item)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if source != "product#P1" || target != "product#P1" {
			t.Errorf("Expected ('product#P1', 'product#P1'), got ('%s', '%s')", source, target)
		}
	})

	t.Run("missing hk", func(t *testing.T) {
		item := Item{
			"sk": &types.AttributeValueMemberS{Value: "product#P1"},
		}

		_, _, err := UnmarshalTableKey(item)
		if err == nil {
			t.Error("Expected error from missing hk")
		}
	})

	t.Run("missing sk", func(t *testing.T) {
		item := Item{
			"hk": &types.AttributeValueMemberS{Value: "product#P1"},
		}

		_, _, err := UnmarshalTableKey(item)
		if err == nil {
			t.Error("Expected error from missing sk")
		}
	})
}

func TestUnmarshalEntity(t *testing.T) {
	t.Run("empty items", func(t *testing.T) {
		var order Order
		_, err := UnmarshalEntity([]Item{}, &order)
		if err != ErrItemNotFound {
			t.Errorf("Expected ErrItemNotFound, got %v", err)
		}
	})

	t.Run("valid entity with relationships", func(t *testing.T) {
		// Create proper entity data
		orderData := &Order{ID: "O1", PurchasedBy: "john"}
		orderDataAttr, err := attributevalue.Marshal(orderData)
		if err != nil {
			t.Fatalf("Failed to marshal order data: %v", err)
		}

		productData := &Product{ID: "P1", Category: "electronics"}
		productDataAttr, err := attributevalue.Marshal(productData)
		if err != nil {
			t.Fatalf("Failed to marshal product data: %v", err)
		}

		selfItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data":  orderDataAttr,
		}

		refItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
			"data":  productDataAttr,
		}

		items := []Item{selfItem, refItem}

		var order Order
		relationships, err := UnmarshalEntity(items, &order)
		if err != nil {
			t.Fatalf("Failed to unmarshal entity: %v", err)
		}

		if len(relationships) != 2 {
			t.Errorf("Expected 2 relationships, got %d", len(relationships))
		}
	})

	t.Run("invalid target key format", func(t *testing.T) {
		selfItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data":  &types.AttributeValueMemberS{Value: "test"},
		}

		invalidRefItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "invalid-format"},
			"label": &types.AttributeValueMemberS{Value: "order/O1/products"},
			"data":  &types.AttributeValueMemberS{Value: "test"},
		}

		items := []Item{selfItem, invalidRefItem}

		var order Order
		_, err := UnmarshalEntity(items, &order)
		if err == nil {
			t.Error("Expected error from invalid target key format")
		}
	})

	t.Run("invalid label format", func(t *testing.T) {
		selfItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data":  &types.AttributeValueMemberS{Value: "test"},
		}

		invalidLabelItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "invalid"},
			"data":  &types.AttributeValueMemberS{Value: "test"},
		}

		items := []Item{selfItem, invalidLabelItem}

		var order Order
		_, err := UnmarshalEntity(items, &order)
		if err == nil {
			t.Error("Expected error from invalid label format")
		}
	})
}

func TestUnmarshalList(t *testing.T) {
	t.Run("valid items", func(t *testing.T) {
		// Create proper entity data
		productData := &Product{ID: "P1", Category: "electronics"}
		dataAttr, err := attributevalue.Marshal(productData)
		if err != nil {
			t.Fatalf("Failed to marshal product data: %v", err)
		}

		items := []Item{
			{
				"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
				"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
				"label": &types.AttributeValueMemberS{Value: "product"},
				"data":  dataAttr,
			},
		}

		products := []Product{}
		relationships, err := UnmarshalList(items, &products)
		if err != nil {
			t.Fatalf("Failed to unmarshal list: %v", err)
		}

		if len(relationships) != 1 {
			t.Errorf("Expected 1 relationship, got %d", len(relationships))
		}
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		items := []Item{
			{
				"hk":    &types.AttributeValueMemberS{Value: "product#P1"},
				"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
				"label": &types.AttributeValueMemberS{Value: "product"},
				"data":  &types.AttributeValueMemberS{Value: "test"},
			},
		}

		products := []*Product{} // Empty slice
		_, err := UnmarshalList(items, &products)
		if err == nil {
			t.Error("Expected error from mismatched lengths")
		}
	})
}
