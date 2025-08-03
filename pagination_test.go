package dynamap

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Tests for pagination functionality

func TestPagination(t *testing.T) {
	table := NewTable("test-table")
	client := newMockDynamoDBClient()
	paginator := table.Paginator(client)
	ctx := context.Background()

	t.Run("nil lastkey returns empty cursor", func(t *testing.T) {
		cursor, err := paginator.PageCursor(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to create cursor: %v", err)
		}
		if cursor != "" {
			t.Errorf("Expected empty cursor for nil lastkey, got %s", cursor)
		}
	})

	t.Run("empty lastkey returns empty cursor", func(t *testing.T) {
		cursor, err := paginator.PageCursor(ctx, Item{})
		if err != nil {
			t.Fatalf("Failed to create cursor: %v", err)
		}
		if cursor != "" {
			t.Errorf("Expected empty cursor for empty lastkey, got %s", cursor)
		}
	})

	t.Run("valid lastkey creates and retrieves cursor", func(t *testing.T) {
		lastkey := Item{
			"hk": &types.AttributeValueMemberS{Value: "test#123"},
			"sk": &types.AttributeValueMemberS{Value: "test#456"},
		}

		cursor, err := paginator.PageCursor(ctx, lastkey)
		if err != nil {
			t.Fatalf("Failed to create cursor: %v", err)
		}
		if cursor == "" {
			t.Error("Expected non-empty cursor for valid lastkey")
		}

		retrievedKey, err := paginator.StartKey(ctx, cursor)
		if err != nil {
			t.Fatalf("Failed to get start key: %v", err)
		}
		if retrievedKey == nil {
			t.Error("Expected non-nil start key")
		}
	})

	t.Run("empty cursor returns nil start key", func(t *testing.T) {
		retrievedKey, err := paginator.StartKey(ctx, "")
		if err != nil {
			t.Fatalf("Failed to get start key: %v", err)
		}
		if retrievedKey != nil {
			t.Error("Expected nil key for empty cursor")
		}
	})

	t.Run("non-existent cursor returns nil", func(t *testing.T) {
		result, err := paginator.StartKey(ctx, "non-existent-cursor")
		if err != nil {
			t.Errorf("Unexpected error for non-existent cursor: %v", err)
		}
		if result != nil {
			t.Error("Expected nil result for non-existent cursor")
		}
	})

	t.Run("cursor with empty key data returns nil", func(t *testing.T) {
		emptyCursor := &PageCursor{
			Cursor: "empty-cursor",
			Key:    []byte{},
		}

		putInput, err := table.MarshalPut(emptyCursor)
		if err != nil {
			t.Fatalf("Failed to marshal empty cursor: %v", err)
		}

		_, err = client.PutItem(ctx, putInput)
		if err != nil {
			t.Fatalf("Failed to store empty cursor: %v", err)
		}

		result, err := paginator.StartKey(ctx, "empty-cursor")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != nil {
			t.Error("Expected nil result for empty key data")
		}
	})
}

func TestMarshalAndUnmarshalStartKey(t *testing.T) {
	table := NewTable("test-table")
	client := newMockDynamoDBClient()
	paginator := table.Paginator(client)
	ctx := context.Background()

	t.Run("marshal and unmarshal valid key", func(t *testing.T) {
		lastKey := Item{
			"hk": &types.AttributeValueMemberS{Value: "test#123"},
			"sk": &types.AttributeValueMemberS{Value: "test#456"},
		}

		cursor, err := MarshalStartKey(ctx, paginator, lastKey)
		if err != nil {
			t.Fatalf("Failed to marshal start key: %v", err)
		}
		if cursor == "" {
			t.Error("Expected non-empty cursor")
		}

		retrievedKey, err := UnmarshalStartKey(ctx, paginator, cursor)
		if err != nil {
			t.Fatalf("Failed to unmarshal start key: %v", err)
		}
		if retrievedKey == nil {
			t.Error("Expected non-nil retrieved key")
		}
	})

	t.Run("marshal nil key returns empty cursor", func(t *testing.T) {
		cursor, err := MarshalStartKey(ctx, paginator, nil)
		if err != nil {
			t.Fatalf("Failed to marshal nil start key: %v", err)
		}
		if cursor != "" {
			t.Error("Expected empty cursor for nil key")
		}
	})

	t.Run("unmarshal empty cursor returns nil", func(t *testing.T) {
		retrievedKey, err := UnmarshalStartKey(ctx, paginator, "")
		if err != nil {
			t.Fatalf("Failed to unmarshal empty cursor: %v", err)
		}
		if retrievedKey != nil {
			t.Error("Expected nil key for empty cursor")
		}
	})
}

func TestPageCursor(t *testing.T) {
	t.Run("MarshalSelf sets correct options", func(t *testing.T) {
		cursor := &PageCursor{Cursor: "test-cursor"}
		opts := &MarshalOptions{}

		err := cursor.MarshalSelf(opts)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if opts.SourcePrefix != "page" {
			t.Errorf("Expected SourcePrefix 'page', got %s", opts.SourcePrefix)
		}
		if opts.SourceID != "test-cursor" {
			t.Errorf("Expected SourceID 'test-cursor', got %s", opts.SourceID)
		}
		if opts.Label != "page" {
			t.Errorf("Expected Label 'page', got %s", opts.Label)
		}
	})
}

func TestGenerateCursor(t *testing.T) {
	t.Run("generates unique cursors", func(t *testing.T) {
		cursors := make(map[string]bool)

		for i := 0; i < 10; i++ {
			cursor, err := generateCursor()
			if err != nil {
				t.Fatalf("Unexpected error on iteration %d: %v", i, err)
			}
			if cursor == "" {
				t.Errorf("Expected non-empty cursor on iteration %d", i)
			}
			if cursors[cursor] {
				t.Errorf("Duplicate cursor generated: %s", cursor)
			}
			cursors[cursor] = true
		}
	})
}
