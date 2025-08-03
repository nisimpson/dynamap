package dynamap

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Test entities that produce errors for testing error paths

type errorEntity struct{}

func (e *errorEntity) MarshalSelf(opts *MarshalOptions) error {
	return errors.New("marshal error")
}

type errorRefEntity struct{}

func (e *errorRefEntity) MarshalSelf(opts *MarshalOptions) error {
	opts.SourcePrefix = "error"
	opts.SourceID = "E1"
	opts.TargetPrefix = "error"
	opts.TargetID = "E1"
	opts.Label = "error"
	return nil
}

func (e *errorRefEntity) MarshalRefs(ctx *RelationshipContext) error {
	return errors.New("marshal refs error")
}

type errorUnmarshalEntity struct{}

func (e *errorUnmarshalEntity) UnmarshalSelf(rel *Relationship) error {
	return errors.New("unmarshal self error")
}

type errorRefUnmarshalEntity struct{}

func (e *errorRefUnmarshalEntity) UnmarshalSelf(rel *Relationship) error {
	return nil // No error for self
}

func (e *errorRefUnmarshalEntity) UnmarshalRef(name string, id string, ref *Relationship) error {
	return errors.New("unmarshal ref error")
}

// Tests for error handling

func TestMarshalRelationshipsErrors(t *testing.T) {
	t.Run("marshal self error", func(t *testing.T) {
		entity := &errorEntity{}

		_, err := MarshalRelationships(entity)
		if err == nil {
			t.Error("Expected error from MarshalRelationships")
		}
	})

	t.Run("marshal refs error", func(t *testing.T) {
		entity := &errorRefEntity{}

		_, err := MarshalRelationships(entity)
		if err == nil {
			t.Error("Expected error from MarshalRefs")
		}
	})
}

func TestRelationshipContextErrors(t *testing.T) {
	ctx := &RelationshipContext{
		source: "test#123",
		opts: MarshalOptions{
			SourcePrefix:   "test",
			SourceID:       "123",
			KeyDelimiter:   "#",
			LabelDelimiter: "/",
			Tick:           DefaultClock,
		},
	}

	t.Run("AddOne with marshal error", func(t *testing.T) {
		ctx.AddOne("error", &errorEntity{})

		if ctx.err == nil {
			t.Error("Expected error in RelationshipContext")
		}
	})

	t.Run("AddOne skips when error exists", func(t *testing.T) {
		// ctx already has an error from previous test
		initialErrCount := len(ctx.refs)
		ctx.AddOne("another", &Product{ID: "P1", Category: "test"})

		// Should not add more refs when error exists
		if len(ctx.refs) != initialErrCount {
			t.Error("Expected AddOne to be skipped when error exists")
		}
	})

	t.Run("AddMany stops on first error", func(t *testing.T) {
		// Reset context
		ctx.err = nil
		ctx.refs = nil

		ctx.AddMany("many", []Marshaler{&errorEntity{}})

		if ctx.err == nil {
			t.Error("Expected error from AddMany")
		}
	})
}

func TestUnmarshalSelfErrors(t *testing.T) {
	t.Run("unmarshal self error", func(t *testing.T) {
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

		errorEntity := &errorUnmarshalEntity{}
		_, err = UnmarshalSelf(item, errorEntity)
		if err == nil {
			t.Error("Expected error from UnmarshalSelf")
		}
	})
}

func TestUnmarshalEntityErrors(t *testing.T) {
	t.Run("unmarshal ref error", func(t *testing.T) {
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

		errorEntity := &errorRefUnmarshalEntity{}
		_, err = UnmarshalEntity(items, errorEntity)
		if err == nil {
			t.Error("Expected error from UnmarshalRef")
		}
	})

	t.Run("invalid relationship item", func(t *testing.T) {
		// Create proper entity data
		orderData := &Order{ID: "O1", PurchasedBy: "john"}
		orderDataAttr, err := attributevalue.Marshal(orderData)
		if err != nil {
			t.Fatalf("Failed to marshal order data: %v", err)
		}

		selfItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data":  orderDataAttr,
		}

		invalidItem := Item{
			"invalid": &types.AttributeValueMemberS{Value: "test"},
		}

		items := []Item{selfItem, invalidItem}

		var order Order
		_, err = UnmarshalEntity(items, &order)
		if err == nil {
			t.Error("Expected error from invalid relationship item")
		}
	})

	t.Run("invalid label format", func(t *testing.T) {
		// Create proper entity data
		orderData := &Order{ID: "O1", PurchasedBy: "john"}
		orderDataAttr, err := attributevalue.Marshal(orderData)
		if err != nil {
			t.Fatalf("Failed to marshal order data: %v", err)
		}

		selfItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"label": &types.AttributeValueMemberS{Value: "order"},
			"data":  orderDataAttr,
		}

		// Create an item with invalid label format (2 parts instead of 1 or 3)
		invalidItem := Item{
			"hk":    &types.AttributeValueMemberS{Value: "order#O1"},
			"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
			"label": &types.AttributeValueMemberS{Value: "order/O1"}, // Only 2 parts
			"data":  orderDataAttr,
		}

		items := []Item{selfItem, invalidItem}

		var order Order
		_, err = UnmarshalEntity(items, &order)
		if err == nil {
			t.Error("Expected error from invalid label format")
		}
	})
}

func TestUnmarshalListErrors(t *testing.T) {
	t.Run("unmarshal self error", func(t *testing.T) {
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

		entities := []errorUnmarshalEntity{}

		_, err = UnmarshalList(items, &entities)
		if err == nil {
			t.Error("Expected error from UnmarshalSelf")
		}
	})

	t.Run("invalid relationship item", func(t *testing.T) {
		items := []Item{
			{
				"invalid": &types.AttributeValueMemberS{Value: "test"},
			},
		}

		entities := []Product{}

		_, err := UnmarshalList(items, &entities)
		if err == nil {
			t.Error("Expected error from invalid relationship item")
		}
	})

	t.Run("invalid attribute type", func(t *testing.T) {
		items := []Item{
			{
				"hk":    &types.AttributeValueMemberN{Value: "123"}, // Number instead of string
				"sk":    &types.AttributeValueMemberS{Value: "product#P1"},
				"label": &types.AttributeValueMemberS{Value: "product"},
				"data":  &types.AttributeValueMemberS{Value: "test"},
			},
		}

		entities := []Product{}

		_, err := UnmarshalList(items, &entities)
		if err == nil {
			t.Error("Expected error from invalid attribute type")
		}
	})
}

func TestTableOperationErrors(t *testing.T) {
	table := NewTable("test-table")

	t.Run("MarshalPut with marshal error", func(t *testing.T) {
		errorEntity := &errorEntity{}
		_, err := table.MarshalPut(errorEntity)
		if err == nil {
			t.Error("Expected error from MarshalPut")
		}
	})

	t.Run("MarshalBatch with marshal error", func(t *testing.T) {
		entity := &errorRefEntity{}
		_, err := table.MarshalBatch(entity)
		if err == nil {
			t.Error("Expected error from MarshalBatch")
		}
	})

	t.Run("MarshalGet with marshal error", func(t *testing.T) {
		errorEntity := &errorEntity{}
		_, err := table.MarshalGet(errorEntity)
		if err == nil {
			t.Error("Expected error from MarshalGet")
		}
	})

	t.Run("MarshalDelete with marshal error", func(t *testing.T) {
		errorEntity := &errorEntity{}
		_, err := table.MarshalDelete(errorEntity)
		if err == nil {
			t.Error("Expected error from MarshalDelete")
		}
	})
}

func TestQueryErrors(t *testing.T) {
	table := NewTable("test-table")

	t.Run("QueryEntity with marshal error", func(t *testing.T) {
		queryEntity := &QueryEntity{
			Source: &errorEntity{},
		}

		_, err := table.MarshalQuery(queryEntity)
		if err == nil {
			t.Error("Expected error from QueryEntity with error entity")
		}
	})
}
