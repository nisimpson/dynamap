package dynamap

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Tests for query functionality

func TestQueryList(t *testing.T) {
	table := NewTable("test-table")

	t.Run("basic query", func(t *testing.T) {
		queryList := &QueryList{
			Label:          "product",
			Limit:          10,
			SortDescending: false,
		}

		queryInput, err := table.MarshalQuery(queryList)
		if err != nil {
			t.Fatalf("Failed to marshal query: %v", err)
		}

		if *queryInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *queryInput.TableName)
		}

		if *queryInput.IndexName != "ref-index" {
			t.Errorf("Expected index name 'ref-index', got %s", *queryInput.IndexName)
		}

		if *queryInput.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", *queryInput.Limit)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		labelSortFilter := expression.Key("gsi1_sk").BeginsWith("electronics")
		conditionFilter := expression.Name("data.category").Equal(expression.Value("electronics"))

		queryList := &QueryList{
			Label:           "product",
			RefSortFilter:   labelSortFilter,
			ConditionFilter: conditionFilter,
			Limit:           10,
			StartKey:        Item{"test": &types.AttributeValueMemberS{Value: "test"}},
			SortDescending:  true,
		}

		opts := NewMarshalOptions()
		input, err := queryList.MarshalQuery(&opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if input == nil {
			t.Error("Expected non-nil input")
		}
	})

	t.Run("without optional filters", func(t *testing.T) {
		queryList := &QueryList{
			Label: "product",
		}

		opts := NewMarshalOptions()
		input, err := queryList.MarshalQuery(&opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if input == nil {
			t.Error("Expected non-nil input")
		}
	})
}

func TestQueryEntity(t *testing.T) {
	table := NewTable("test-table")
	order := &Order{ID: "O1", PurchasedBy: "john"}

	t.Run("basic query", func(t *testing.T) {
		queryEntity := &QueryEntity{
			Source:         order,
			Limit:          20,
			SortDescending: true,
		}

		queryInput, err := table.MarshalQuery(queryEntity)
		if err != nil {
			t.Fatalf("Failed to marshal query: %v", err)
		}

		if *queryInput.TableName != "test-table" {
			t.Errorf("Expected table name 'test-table', got %s", *queryInput.TableName)
		}

		if queryInput.IndexName != nil {
			t.Errorf("Expected no index name for QueryEntity, got %s", *queryInput.IndexName)
		}

		if *queryInput.Limit != 20 {
			t.Errorf("Expected limit 20, got %d", *queryInput.Limit)
		}

		if *queryInput.ScanIndexForward != false {
			t.Errorf("Expected ScanIndexForward false, got %t", *queryInput.ScanIndexForward)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		targetFilter := expression.Key("sk").BeginsWith("product#")
		conditionFilter := expression.Name("data.category").Equal(expression.Value("electronics"))

		queryEntity := &QueryEntity{
			Source:          &Product{ID: "P1", Category: "electronics"},
			TargetFilter:    targetFilter,
			ConditionFilter: conditionFilter,
			Limit:           20,
			StartKey:        Item{"test": &types.AttributeValueMemberS{Value: "test"}},
			SortDescending:  true,
		}

		opts := NewMarshalOptions()
		input, err := queryEntity.MarshalQuery(&opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if input == nil {
			t.Error("Expected non-nil input")
		}
	})

	t.Run("without optional filters", func(t *testing.T) {
		queryEntity := &QueryEntity{
			Source: &Product{ID: "P1", Category: "electronics"},
		}

		opts := NewMarshalOptions()
		input, err := queryEntity.MarshalQuery(&opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if input == nil {
			t.Error("Expected non-nil input")
		}
	})
}

func TestQueryUseRefIndex(t *testing.T) {
	table := Table{RefIndexName: "index"}
	t.Run("QueryList uses ref index", func(t *testing.T) {
		queryList := &QueryList{Label: "product"}
		if queryList.UseIndex(&table) != "index" {
			t.Error("Expected QueryList to use ref index")
		}
	})

	t.Run("QueryEntity does not use ref index", func(t *testing.T) {
		queryEntity := &QueryEntity{Source: &Product{ID: "P1", Category: "electronics"}}
		if queryEntity.UseIndex(&table) != "" {
			t.Error("Expected QueryEntity to not use ref index")
		}
	})
}
func TestFilterFunctions(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("PeriodBefore", func(t *testing.T) {
		condition := PeriodBefore("test_field", testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("PeriodAfter", func(t *testing.T) {
		condition := PeriodAfter("test_field", testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("PeriodBetween", func(t *testing.T) {
		start := testTime
		end := testTime.Add(time.Hour)
		condition := PeriodBetween("test_field", start, end)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("CreatedBefore", func(t *testing.T) {
		condition := CreatedBefore(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("CreatedAfter", func(t *testing.T) {
		condition := CreatedAfter(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("CreatedBetween", func(t *testing.T) {
		start := testTime
		end := testTime.Add(time.Hour)
		condition := CreatedBetween(start, end)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("MinAge", func(t *testing.T) {
		condition := MinAge(24 * time.Hour)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("MaxAge", func(t *testing.T) {
		condition := MaxAge(24 * time.Hour)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("UpdatedBefore", func(t *testing.T) {
		condition := UpdatedBefore(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("UpdatedAfter", func(t *testing.T) {
		condition := UpdatedAfter(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("UpdatedBetween", func(t *testing.T) {
		start := testTime
		end := testTime.Add(time.Hour)
		condition := UpdatedBetween(start, end)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("ExpiresAfter", func(t *testing.T) {
		condition := ExpiresAfter(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("ExpiresBefore", func(t *testing.T) {
		condition := ExpiresBefore(testTime)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})

	t.Run("ExpiresIn", func(t *testing.T) {
		condition := ExpiresIn(24 * time.Hour)
		if !condition.IsSet() {
			t.Error("Expected condition to be set")
		}
	})
}
