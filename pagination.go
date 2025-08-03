package dynamap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func init() {
	// Register DynamoDB types with gob
	gob.Register(map[string]types.AttributeValue{})
	gob.Register(&types.AttributeValueMemberS{})
	gob.Register(&types.AttributeValueMemberN{})
	gob.Register(&types.AttributeValueMemberB{})
	gob.Register(&types.AttributeValueMemberSS{})
	gob.Register(&types.AttributeValueMemberNS{})
	gob.Register(&types.AttributeValueMemberBS{})
	gob.Register(&types.AttributeValueMemberM{})
	gob.Register(&types.AttributeValueMemberL{})
	gob.Register(&types.AttributeValueMemberNULL{})
	gob.Register(&types.AttributeValueMemberBOOL{})
}

// Paginator handles pagination by converting last evaluated keys into string
// cursors for clients, and in turn converting client cursors into start keys
// to continue paging of query results.
type Paginator interface {
	// PageCursor generates a string token from the provided start key. Implementors
	// should return an empty token if the start key is nil or empty.
	PageCursor(ctx context.Context, lastkey Item) (string, error)
	// StartKey generates a dynamodb start key from the provided cursor. Implementors
	// should return a nil item if the cursor is an empty string.
	StartKey(ctx context.Context, cursor string) (Item, error)
}

// TablePaginator implements Pagination by storing and retrieving start keys in the same table.
type TablePaginator struct {
	table  *Table         // table configuration
	client DynamoDBClient // dynamodb client
}

// PageCursor represents an item in the dynamodb table that stores last evaluated key
// information from query results. Cursor is generated from the current time and salt
// while Key is a gob encoded form of the last evaluated key.
//
// PageCursor implements Marshaler and Unmarshaler.
type PageCursor struct {
	Cursor string
	Key    []byte
}

// MarshalSelf implements Marshaler by providing a self-relationship:
//   - source id: current time + salt
//   - source prefix: "page"
//   - ttl: 24 hours (default)
func (p *PageCursor) MarshalSelf(opts *MarshalOptions) error {
	opts.SourcePrefix = "page"
	opts.SourceID = p.Cursor
	opts.TargetPrefix = "page"
	opts.TargetID = p.Cursor
	opts.Label = "page"
	opts.TimeToLive = 24 * time.Hour // Default TTL, can be overridden by table config
	opts.RefSortKey = p.Cursor

	return nil
}

// PageCursor implements Pagination by storing the last evaluated key into the dynamodb table.
// The key itself is stored as a self-relationship, with the relationship data encoded as
// binary. If lastkey is nil, an empty string is returned.
func (t *TablePaginator) PageCursor(ctx context.Context, lastkey Item) (string, error) {
	if lastkey == nil || len(lastkey) == 0 {
		return "", nil
	}

	// Generate a unique cursor ID
	cursor, err := generateCursor()
	if err != nil {
		return "", fmt.Errorf("failed to generate cursor: %w", err)
	}

	// Convert DynamoDB types to JSON for storage
	keyData, err := attributevalue.MarshalMap(lastkey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal last key: %w", err)
	}

	// Encode as gob
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(keyData); err != nil {
		return "", fmt.Errorf("failed to encode last key: %w", err)
	}

	// Create the page cursor
	pageCursor := &PageCursor{
		Cursor: cursor,
		Key:    buf.Bytes(),
	}

	// Store the cursor in the table with TTL
	putInput, err := t.table.MarshalPut(pageCursor, func(opts *MarshalOptions) {
		opts.TimeToLive = t.table.PaginationTTL
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal page cursor: %w", err)
	}

	// Store in DynamoDB
	_, err = t.client.PutItem(ctx, putInput)
	if err != nil {
		return "", fmt.Errorf("failed to store page cursor: %w", err)
	}

	return cursor, nil
}

// StartKey implements Pagination by retrieving the self-relationship referenced by
// cursor. If found the PageCursor data is decoded from binary and returned.
// If the relationship is not found, nil is returned.
func (t *TablePaginator) StartKey(ctx context.Context, cursor string) (Item, error) {
	if cursor == "" {
		return nil, nil
	}

	// Create a page cursor to get the key
	pageCursor := &PageCursor{Cursor: cursor}

	// Get the cursor from the table
	getInput, err := t.table.MarshalGet(pageCursor)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	result, err := t.client.GetItem(ctx, getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get page cursor: %w", err)
	}

	if result.Item == nil {
		// Cursor not found or expired
		return nil, nil
	}

	// Unmarshal the cursor
	_, err = UnmarshalSelf(result.Item, pageCursor)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal page cursor: %w", err)
	}

	// Decode the key data
	if len(pageCursor.Key) == 0 {
		return nil, nil
	}

	buf := bytes.NewBuffer(pageCursor.Key)
	decoder := gob.NewDecoder(buf)

	var keyData map[string]types.AttributeValue
	if err := decoder.Decode(&keyData); err != nil {
		return nil, fmt.Errorf("failed to decode last key: %w", err)
	}

	return keyData, nil
}

// Paginator returns a Paginator to extract and generate client cursors.
func (t *Table) Paginator(client DynamoDBClient) Paginator {
	return &TablePaginator{
		table:  t,
		client: client,
	}
}

// MarshalStartKey marshals a page key into a page cursor to return to clients.
func MarshalStartKey(ctx context.Context, p Paginator, lastkey Item) (string, error) {
	return p.PageCursor(ctx, lastkey)
}

// UnmarshalStartKey unmarshals a page key from the provided cursor.
func UnmarshalStartKey(ctx context.Context, p Paginator, cursor string) (Item, error) {
	return p.StartKey(ctx, cursor)
}

// generateCursor creates a unique cursor string using current time and random bytes
func generateCursor() (string, error) {
	// Use current time in nanoseconds for uniqueness
	timestamp := time.Now().UnixNano()

	// Add some random bytes for additional uniqueness
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Combine timestamp and random bytes
	combined := fmt.Sprintf("%d_%s", timestamp, base64.URLEncoding.EncodeToString(randomBytes))

	// Encode as base64 for URL safety
	return base64.URLEncoding.EncodeToString([]byte(combined)), nil
}
