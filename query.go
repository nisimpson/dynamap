package dynamap

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// QueryMarshaler can marshal input into a dynamodb query request.
type QueryMarshaler interface {
	// MarshalQuery marshals the query into a DynamoDB QueryInput with the given options.
	MarshalQuery(*MarshalOptions) (*dynamodb.QueryInput, error)
	// UseIndex returns the index name to use for the query, or empty string for the main table.
	UseIndex(*Table) string
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

func (QueryEntity) UseIndex(*Table) string { return "" }
func (QueryList) UseIndex(t *Table) string { return t.RefIndexName }

// PeriodBefore creates a condition that filters for timestamps before or equal to the given moment.
func PeriodBefore(name string, moment time.Time) expression.ConditionBuilder {
	value := moment.Format(time.RFC3339)
	return expression.Name(name).LessThanEqual(expression.Value(value))
}

// PeriodAfter creates a condition that filters for timestamps after or equal to the given moment.
func PeriodAfter(name string, moment time.Time) expression.ConditionBuilder {
	value := moment.Format(time.RFC3339)
	return expression.Name(name).GreaterThanEqual(expression.Value(value))
}

// PeriodBetween creates a condition that filters for timestamps between the start and end times.
func PeriodBetween(name string, start, end time.Time) expression.ConditionBuilder {
	startValue := start.Format(time.RFC3339)
	endValue := end.Format(time.RFC3339)
	return expression.Name(name).Between(expression.Value(startValue), expression.Value(endValue))
}

// CreatedBefore creates a condition that filters for entities created before or equal to the given moment.
func CreatedBefore(moment time.Time) expression.ConditionBuilder {
	return PeriodBefore(AttributeNameCreated, moment)
}

// CreatedAfter creates a condition that filters for entities created after or equal to the given moment.
func CreatedAfter(moment time.Time) expression.ConditionBuilder {
	return PeriodAfter(AttributeNameCreated, moment)
}

// CreatedBetween creates a condition that filters for entities created between the start and end times.
func CreatedBetween(start, end time.Time) expression.ConditionBuilder {
	return PeriodBetween(AttributeNameCreated, start, end)
}

// MinAge creates a condition that filters for entities older than the specified age.
func MinAge(age time.Duration) expression.ConditionBuilder {
	var (
		now  = time.Now().UTC()
		then = now.Add(-age)
	)
	return CreatedBefore(then)
}

// MaxAge creates a condition that filters for entities newer than the specified age.
func MaxAge(age time.Duration) expression.ConditionBuilder {
	var (
		now  = time.Now().UTC()
		then = now.Add(-age)
	)
	return CreatedAfter(then)
}

// UpdatedBefore creates a condition that filters for entities updated before or equal to the given moment.
func UpdatedBefore(moment time.Time) expression.ConditionBuilder {
	return PeriodBefore(AttributeNameUpdated, moment)
}

// UpdatedAfter creates a condition that filters for entities updated after or equal to the given moment.
func UpdatedAfter(moment time.Time) expression.ConditionBuilder {
	return PeriodAfter(AttributeNameUpdated, moment)
}

// UpdatedBetween creates a condition that filters for entities updated between the start and end times.
func UpdatedBetween(start, end time.Time) expression.ConditionBuilder {
	return PeriodBetween(AttributeNameUpdated, start, end)
}

// ExpiresAfter creates a condition that filters for entities that expire after the given moment.
func ExpiresAfter(moment time.Time) expression.ConditionBuilder {
	return expression.LessThan(
		expression.Name(AttributeNameExpires),
		expression.Value(moment.Unix()),
	)
}

// ExpiresBefore creates a condition that filters for entities that expire before the given moment.
func ExpiresBefore(moment time.Time) expression.ConditionBuilder {
	return expression.GreaterThan(
		expression.Name(AttributeNameExpires),
		expression.Value(moment.Unix()),
	)
}

// ExpiresIn creates a condition that filters for entities that expire within the specified period.
func ExpiresIn(period time.Duration) expression.ConditionBuilder {
	var (
		now  = time.Now().UTC()
		then = now.Add(period)
	)
	return expression.Between(
		expression.Name(AttributeNameExpires),
		expression.Value(now.Unix()),
		expression.Value(then.Unix()),
	)
}
