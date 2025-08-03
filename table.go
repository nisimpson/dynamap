package dynamap

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	// MaxBatchSize is the maximum number of items allowed in a DynamoDB batch operation.
	MaxBatchSize = 25
)

// MarshalPut marshals the input into a dynamodb put item input request. The request will
// contain the entity's self-relationship; to marshal all entity relationships, use the
// MarshalBatch function.
func (t *Table) MarshalPut(in Marshaler, opts ...func(*MarshalOptions)) (*dynamodb.PutItemInput, error) {
	// Marshal relationships (will only contain self due to SkipRefs)
	relationships, err := MarshalRelationships(in, func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only marshal self for put operations
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relationships: %w", err)
	}

	if len(relationships) != 1 {
		return nil, fmt.Errorf("expected exactly 1 relationship for put, got %d", len(relationships))
	}

	// Marshal the relationship to DynamoDB item
	item, err := attributevalue.MarshalMap(relationships[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item: %w", err)
	}

	return &dynamodb.PutItemInput{
		TableName: aws.String(t.TableName),
		Item:      item,
	}, nil
}

// MarshalBatch marshals the input into multiple batch write put requests. Since there is a
// limit on how many requests can be contained in a single input, the requests are chunked
// in sizes of 25 or less.
func (t *Table) MarshalBatch(in RefMarshaler, opts ...func(*MarshalOptions)) ([]*dynamodb.BatchWriteItemInput, error) {
	// Marshal all relationships
	relationships, err := MarshalRelationships(in, func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = false // include all relationships for batch operations
	})

	if err != nil {
		return nil, fmt.Errorf("failed to marshal relationships: %w", err)
	}

	// Chunk relationships into batches
	var batches []*dynamodb.BatchWriteItemInput

	for i := 0; i < len(relationships); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(relationships) {
			end = len(relationships)
		}

		var writeRequests []types.WriteRequest
		for _, rel := range relationships[i:end] {
			item, err := attributevalue.MarshalMap(rel)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal relationship: %w", err)
			}

			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{Item: item},
			})
		}

		batch := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				t.TableName: writeRequests,
			},
		}
		batches = append(batches, batch)
	}

	return batches, nil
}

// MarshalGet marshals the input into a get item request. The self relationship key is used
// to retrieve the relationship from dynamodb.
func (t *Table) MarshalGet(in Marshaler, opts ...func(*MarshalOptions)) (*dynamodb.GetItemInput, error) {
	// Create marshal options with table defaults
	marshalOpts := newMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only need self relationship for key
	})

	// Marshal to get the key information
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	// Create the key
	sourceKey := marshalOpts.sourceKey()
	key := map[string]types.AttributeValue{
		"hk": &types.AttributeValueMemberS{Value: sourceKey},
		"sk": &types.AttributeValueMemberS{Value: sourceKey},
	}

	return &dynamodb.GetItemInput{
		TableName: aws.String(t.TableName),
		Key:       key,
	}, nil
}

// MarshalDelete marshals the input into a delete item request.
// The self relationship key is used to retrieve the relationship from dynamodb.
func (t *Table) MarshalDelete(in Marshaler, opts ...func(*MarshalOptions)) (*dynamodb.DeleteItemInput, error) {
	// Create marshal options with table defaults
	marshalOpts := newMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only need self relationship for key
	})

	// Marshal to get the key information
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	// Create the key
	sourceKey := marshalOpts.sourceKey()
	key := map[string]types.AttributeValue{
		"hk": &types.AttributeValueMemberS{Value: sourceKey},
		"sk": &types.AttributeValueMemberS{Value: sourceKey},
	}

	return &dynamodb.DeleteItemInput{
		TableName: aws.String(t.TableName),
		Key:       key,
	}, nil
}

// MarshalQuery marshals the input into a query item request
func (t *Table) MarshalQuery(in QueryMarshaler, opts ...func(*MarshalOptions)) (*dynamodb.QueryInput, error) {
	// Create marshal options with table defaults
	marshalOpts := newMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.apply(opts)
	})

	// Marshal the query
	input, err := in.MarshalQuery(&marshalOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Set the table name
	input.TableName = aws.String(t.TableName)

	// Set the index name if this is a QueryList (queries on label)
	if in.UseRefIndex() {
		input.IndexName = aws.String(t.RefIndexName)
	}

	return input, nil
}
