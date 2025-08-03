package dynamap

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// QueryMarshaler can marshal input into a dynamodb query request.
type QueryMarshaler interface {
	MarshalQuery(*MarshalOptions) (*dynamodb.QueryInput, error)
	UseRefIndex() bool
}

// QueryList is a QueryMarshaler that searches the table for collections
// of entities with a specific label.
type QueryList struct {
	Label           string                         // The relationship label
	RefSortFilter   expression.KeyConditionBuilder // Optional filters on the label sort key
	ConditionFilter expression.ConditionBuilder    // Optional filters on the relationship
	Limit           int                            // Maximum number of items to return
	StartKey        Item                           // Exclusive start key for pagination
	SortDescending  bool                           // Scan direction (default: false)
}

// MarshalQuery implements QueryMarshaler for QueryList.
func (q *QueryList) MarshalQuery(opts *MarshalOptions) (*dynamodb.QueryInput, error) {
	// Build the key condition for the label
	keyCondition := expression.Key(AttributeNameLabel).Equal(expression.Value(q.Label))

	// Add label sort filter if provided
	if q.RefSortFilter.IsSet() {
		keyCondition = keyCondition.And(q.RefSortFilter)
	}

	// Build the expression
	builder := expression.NewBuilder().WithKeyCondition(keyCondition)

	// Add condition filter if provided
	if q.ConditionFilter.IsSet() {
		builder = builder.WithFilter(q.ConditionFilter)
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(!q.SortDescending),
	}

	// Add filter expression if present
	if q.ConditionFilter.IsSet() {
		input.FilterExpression = expr.Filter()
	}

	// Add limit if specified
	if q.Limit > 0 {
		input.Limit = aws.Int32(int32(q.Limit))
	}

	// Add start key if provided
	if q.StartKey != nil {
		input.ExclusiveStartKey = q.StartKey
	}

	return input, nil
}

// QueryEntity is a QueryMarshaler that searches within an entity's partition for
// key relationships. The results of this query should be unmarshaled with
// UnmarshalEntity.
type QueryEntity struct {
	Source          Marshaler                      // The source entity
	TargetFilter    expression.KeyConditionBuilder // Optional filters on the table sort key
	ConditionFilter expression.ConditionBuilder    // Optional filters on the relationship
	Limit           int                            // Maximum number of items to return
	StartKey        Item                           // Exclusive start key for pagination
	SortDescending  bool                           // If true, scans backward
}

// MarshalQuery implements QueryMarshaler for QueryEntity.
func (q *QueryEntity) MarshalQuery(opts *MarshalOptions) (*dynamodb.QueryInput, error) {
	// Marshal the source entity to get the key
	sourceOpts := *opts
	sourceOpts.SkipRefs = true

	if err := q.Source.MarshalSelf(&sourceOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal source: %w", err)
	}

	// Create the source key
	sourceKey := opts.sourceKey()

	// Build the key condition for the source
	keyCondition := expression.Key(AttributeNameSource).Equal(expression.Value(sourceKey))

	// Add target filter if provided
	if q.TargetFilter.IsSet() {
		keyCondition = keyCondition.And(q.TargetFilter)
	}

	// Build the expression
	builder := expression.NewBuilder().WithKeyCondition(keyCondition)

	// Add condition filter if provided
	if q.ConditionFilter.IsSet() {
		builder = builder.WithFilter(q.ConditionFilter)
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.QueryInput{
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		ScanIndexForward:          aws.Bool(!q.SortDescending),
	}

	// Add limit if specified
	if q.Limit > 0 {
		input.Limit = aws.Int32(int32(q.Limit))
	}

	// Add start key if provided
	if q.StartKey != nil {
		input.ExclusiveStartKey = q.StartKey
	}

	return input, nil
}

func (QueryEntity) UseRefIndex() bool { return false }
func (QueryList) UseRefIndex() bool   { return true }
