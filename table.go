package dynamap

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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
	marshalOpts := NewMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only need self relationship for key
	})

	// Marshal to get the key information
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	return &dynamodb.GetItemInput{
		TableName: aws.String(t.TableName),
		Key:       marshalOpts.itemKey(),
	}, nil
}

// MarshalDelete marshals the input into a delete item request.
// The self relationship key is used to retrieve the relationship from dynamodb.
func (t *Table) MarshalDelete(in Marshaler, opts ...func(*MarshalOptions)) (*dynamodb.DeleteItemInput, error) {
	// Create marshal options with table defaults
	marshalOpts := NewMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only need self relationship for key
	})

	// Marshal to get the key information
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	return &dynamodb.DeleteItemInput{
		TableName: aws.String(t.TableName),
		Key:       marshalOpts.itemKey(),
	}, nil
}

// Updater can build update expressions for modifying relationships.
type Updater interface {
	// UpdateRelationship builds an update expression using the provided base builder.
	UpdateRelationship(base expression.UpdateBuilder) expression.UpdateBuilder
}

// DataAttribute creates a DynamoDB expression NameBuilder for accessing data attributes within the 'data' field.
// It takes a data attribute suffix and returns a NameBuilder that references 'data.<suffix>'.
// This is used when building expressions to access or modify nested data attributes in DynamoDB items.
//
// Example:
//
//	// Access the 'name' field within the 'data' attribute
//	nameAttr := DataAttribute("name")
//	// Results in expression referencing 'data.name'
func DataAttribute(suffix string) expression.NameBuilder {
	return expression.Name(fmt.Sprintf("%s.%s", AttributeNameData, suffix))
}

// MarshalUpdate marshals the input into a DynamoDB UpdateItem request using the provided updater.
func (t *Table) MarshalUpdate(in Marshaler, updater Updater, opts ...func(*MarshalOptions)) (*dynamodb.UpdateItemInput, error) {
	if updater == nil {
		return nil, fmt.Errorf("updater is required")
	}

	// Create marshal options with table defaults
	marshalOpts := NewMarshalOptions(func(mo *MarshalOptions) {
		mo.KeyDelimiter = t.KeyDelimiter
		mo.LabelDelimiter = t.LabelDelimiter
		mo.apply(opts)
		mo.SkipRefs = true // Only need self relationship for key
	})

	// Marshal to get the key information
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	// Marshal the update expression
	update := expression.Set(
		expression.Name(AttributeNameUpdated),
		expression.Value(marshalOpts.Tick().UTC().Format(time.RFC3339)),
	)
	update = updater.UpdateRelationship(update)
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build update expression: %w", err)
	}

	return &dynamodb.UpdateItemInput{
		TableName:                 aws.String(t.TableName),
		Key:                       marshalOpts.itemKey(),
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              types.ReturnValueUpdatedNew,
	}, nil
}

// MarshalQuery marshals the input into a query item request.
func (t *Table) MarshalQuery(in QueryMarshaler, opts ...func(*MarshalOptions)) (*dynamodb.QueryInput, error) {
	// Create marshal options with table defaults
	marshalOpts := NewMarshalOptions(func(mo *MarshalOptions) {
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
	if index := in.UseIndex(t); index != "" {
		input.IndexName = aws.String(index)
	}

	return input, nil
}
