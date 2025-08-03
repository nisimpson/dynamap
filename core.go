// Package dynamap provides a lightweight entity-relationship abstraction layer
// over the AWS SDK for Go v2 DynamoDB client.
package dynamap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ErrItemNotFound is returned when an item is not found in DynamoDB operations.
var ErrItemNotFound = errors.New("item not found")

// Clock is a function type that returns the current time for dependency injection.
type Clock func() time.Time

// DefaultClock returns the current UTC time.
func DefaultClock() time.Time {
	return time.Now().UTC()
}

// Table contains DynamoDB table configuration and marshal options.
type Table struct {
	TableName      string        // Main table name
	RefIndexName   string        // Ref index name (maps to gsi1_sk attribute)
	KeyDelimiter   string        // Delimiter for hash and sort keys. Default is '#'.
	LabelDelimiter string        // Delimiter for label index hash keys. Default is '/'.
	PaginationTTL  time.Duration // TTL for pagination cursors stored in table
}

// NewTable creates a new Table with default configuration.
func NewTable(tableName string) *Table {
	return &Table{
		TableName:      tableName,
		RefIndexName:   "ref-index",
		KeyDelimiter:   "#",
		LabelDelimiter: "/",
		PaginationTTL:  24 * time.Hour,
	}
}

// MarshalOptions contains configuration options for marshaling entities to relationships.
type MarshalOptions struct {
	SourceID       string        // The entity source identifier
	SourcePrefix   string        // The entity source prefix, usually the entity type
	TargetID       string        // The entity target identifier
	TargetPrefix   string        // The entity target prefix, usually the entity type
	TimeToLive     time.Duration // The lifetime of the relationship
	Label          string        // The relationship label
	Created        time.Time     // Creation timestamp
	Updated        time.Time     // Modification timestamp
	RefSortKey     string        // String that uniquely identifies this relationship on the label index
	Tick           Clock         // Function to get current time for timestamps
	KeyDelimiter   string        // Delimiter to join id and prefix into hash and sort keys
	LabelDelimiter string        // Delimiter to join label segments
	SkipRefs       bool          // If true, relationships will not be marshaled.
}

// WithSelfTarget configures the MarshalOptions for a self-referential relationship.
// This is used when an entity references itself, such as for storing the entity's own data.
//
// Parameters:
//   - label: The prefix/type of the entity (e.g. "user", "order")
//   - id: The unique identifier for the entity
//
// Returns the modified MarshalOptions for method chaining.
func (mo *MarshalOptions) WithSelfTarget(label, id string) *MarshalOptions {
	mo.SourceID = id
	mo.TargetID = id
	mo.SourcePrefix = label
	mo.TargetPrefix = label
	mo.Label = label
	return mo
}

func (mo *MarshalOptions) apply(opts []func(*MarshalOptions)) {
	for _, opt := range opts {
		opt(mo)
	}
}

func (mo MarshalOptions) sourceKey() string {
	return mo.SourcePrefix + mo.KeyDelimiter + mo.SourceID
}

func (mo MarshalOptions) targetKey() string {
	return mo.TargetPrefix + mo.KeyDelimiter + mo.TargetID
}

func (mo MarshalOptions) refLabel(name string) string {
	// label format: <source_prefix>/<source_id>/<relationship_name>
	return mo.SourcePrefix + mo.LabelDelimiter + mo.SourceID + mo.LabelDelimiter + name
}

func (mo MarshalOptions) splitLabel(rel Relationship) (prefix, id, name string, err error) {
	parts := strings.Split(rel.Label, mo.LabelDelimiter)
	if len(parts) == 1 {
		return parts[0], "", "", nil
	} else if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid label length; should be 1 or 3")
	}
	return parts[0], parts[1], parts[2], nil
}

func newMarshalOptions(opts ...func(*MarshalOptions)) MarshalOptions {
	options := MarshalOptions{
		Tick:           DefaultClock,
		KeyDelimiter:   "#",
		LabelDelimiter: "/",
	}
	options.Created = options.Tick()
	options.Updated = options.Tick()
	options.apply(opts)
	return options
}

// Relationship represents an association between two entities. Relationships are
// composed of a source key and a target key, along with a label that categorizes
// the relationship. A "self" relationship has equivalent source and target keys.
// The label format is typically based on the relationship type:
//
//	// self relationship
//	"<source_prefix>"
//	// other relationships
//	"<source_prefix>/<source_id>/<relationship_name>"
//
// For example, for an Order O1 entity with Product P1, P2 relationships:
//
//	| hk       | sk         | label             |
//	| ======== | ========== | ================= |
//	| order#O1 | order#O1   | order             |
//	| order#O1 | product#P1 | order/O1/products |
//	| order#O1 | product#P2 | order/O1/products |
//
// This format allows for the following queries:
//   - To query all relationships on O1, the client would use the hash key "order#O1"
//   - To query all orders, the client would use the label "order" on the ref label GSI
//   - To query all products in an order the use the label "order/O1/products" on the ref
//     label GSI
//
// Relationship also supports create/update timestamps and optional time-to-live attributes.
type Relationship struct {
	Source    string    `dynamodbav:"hk"`                // The source entity (prefix + id)
	Target    string    `dynamodbav:"sk"`                // The target entity (prefix + id)
	Label     string    `dynamodbav:"label"`             // The label, which identifies the type or relationship
	CreatedAt time.Time `dynamodbav:"created_at"`        // creation timestamp
	UpdatedAt time.Time `dynamodbav:"updated_at"`        // modification timestamp
	Expires   time.Time `dynamodbav:"expires,unixtime"`  // time-to-live attribute
	Data      any       `dynamodbav:"data,omitempty"`    // relationship data
	GSI1SK    string    `dynamodbav:"gsi1_sk,omitempty"` // sort index for the ref index
}

const (
	AttributeNameSource     = "hk"
	AttributeNameTarget     = "sk"
	AttributeNameLabel      = "label"
	AttributeNameCreated    = "created_at"
	AttributeNameUpdated    = "updated_at"
	AttributeNameExpires    = "expires"
	AttributeNameData       = "data"
	AttributeNameRefSortKey = "gsi1_sk"
)

func NewRelationship(data any, opts MarshalOptions) Relationship {
	// Set timestamps if not provided
	if opts.Created.IsZero() {
		opts.Created = opts.Tick()
	}
	if opts.Updated.IsZero() {
		opts.Updated = opts.Tick()
	}

	// Create relationship
	rel := Relationship{
		Source:    opts.sourceKey(),
		Target:    opts.targetKey(),
		Label:     opts.Label,
		CreatedAt: opts.Created,
		UpdatedAt: opts.Updated,
		Data:      data, // Store the entity data in the self relationship
		GSI1SK:    opts.RefSortKey,
	}

	if opts.TimeToLive > 0 {
		rel.Expires = opts.Created.Add(opts.TimeToLive)
	}

	return rel
}

// Marshaler can marshal itself into relationship options.
type Marshaler interface {
	// MarshalSelf is invoked by the [MarshalRelationships] function.
	// Implementers should adjust the provided options to set the fields of Relationship.
	MarshalSelf(*MarshalOptions) error
}

// RefMarshaler is a Marshaler that can also marshal its relationships.
type RefMarshaler interface {
	Marshaler
	// MarshalRefs is invoked by the [MarshalRelationships] function.
	// Implementers should add entity relationships via the [RelationshipContext.AddOne]
	// and [RelationshipContext.AddMany] functions for "to-one" and "to-many" relationships, respectively.
	MarshalRefs(*RelationshipContext) error
}

// RelationshipContext provides context for creating relationships from a specific source.
type RelationshipContext struct {
	source string         // Private field to store the source key
	opts   MarshalOptions // Private options for marshaling relationships
	refs   []Relationship // Private field to store accumulated relationships
	err    error          // Private error that occurred during marshaling
}

// Ref represents a simple relationship reference between two entities.
type Ref struct {
	Name     string // Name is the name of the relationship (e.g. "products", "orders")
	SourceID string // SourceID is the identifier of the source entity
	TargetID string // TargetID is the identifier of the target entity
}

// AddOne adds a "to-one" [Relationship] to the context.
func (r *RelationshipContext) AddOne(name string, ref Marshaler) {
	if r.err != nil {
		return // Don't continue if there's already an error
	}

	// Create options for the reference
	refOpts := r.opts

	// Marshal the reference to get its target information
	if err := ref.MarshalSelf(&refOpts); err != nil {
		r.err = fmt.Errorf("failed to marshal reference %s: %w", name, err)
		return
	}

	// Create the relationship with the correct label
	refOpts.SourceID = r.opts.SourceID
	refOpts.SourcePrefix = r.opts.SourcePrefix

	rel := NewRelationship(
		Ref{
			SourceID: r.opts.SourceID,
			TargetID: refOpts.TargetID,
			Name:     name,
		},
		refOpts,
	)

	rel.Source = r.source
	rel.Label = refOpts.refLabel(name)
	r.refs = append(r.refs, rel)
}

// AddMany adds "to-many" [Relationship] items to the context.
func (r *RelationshipContext) AddMany(name string, refs []Marshaler) {
	for _, ref := range refs {
		r.AddOne(name, ref)
		if r.err != nil {
			return // Stop on first error
		}
	}
}

// SliceOf is a convenience function for converting marshalers of a specific
// type into a slice of [Marshaler].
func SliceOf[T Marshaler](in ...T) []Marshaler {
	result := make([]Marshaler, len(in))
	for i, item := range in {
		result[i] = item
	}
	return result
}

// MarshalRelationships marshals the input into a list of relationships. The successful
// result of this function will always contain at least one Relationship, which represents
// the self relationship of the entity. If in is a RefMarshaler, then the result will contain
// additional "to-one" and "to-many" relationships.
func MarshalRelationships(in Marshaler, opts ...func(*MarshalOptions)) ([]Relationship, error) {
	// Create default options
	marshalOpts := newMarshalOptions(opts...)

	// Marshal self relationship
	if err := in.MarshalSelf(&marshalOpts); err != nil {
		return nil, fmt.Errorf("failed to marshal self: %w", err)
	}

	self := NewRelationship(in, marshalOpts)
	relationships := []Relationship{self}

	// If it's a RefMarshaler and we're not skipping refs, marshal relationships
	if refMarshaler, ok := in.(RefMarshaler); ok && !marshalOpts.SkipRefs {
		ctx := &RelationshipContext{
			source: marshalOpts.sourceKey(),
			opts:   marshalOpts,
		}

		if err := refMarshaler.MarshalRefs(ctx); err != nil {
			return nil, fmt.Errorf("failed to marshal refs: %w", err)
		}

		if ctx.err != nil {
			return nil, ctx.err
		}

		relationships = append(relationships, ctx.refs...)
	}

	return relationships, nil
}

// Item is an alias for the dynamodb attribute value map.
type Item = map[string]types.AttributeValue

// Unmarshaler can extract data about itself from the provided Relationship.
type Unmarshaler interface {
	// UnmarshalSelf is invoked by [UnmarshalSelf]. Implementors can extract
	// additional information that was stored in the provided Relationship.
	UnmarshalSelf(*Relationship) error
}

// RefUnmarshaler can extract data about both itself and any associated relationships.
type RefUnmarshaler interface {
	// UnmarshalRef is invoked by [UnmarshalEntity] Implementers can extract
	// additional information that was stored in the provided Relationship.
	// For convenience, the relationship name and identifier are provided.
	UnmarshalRef(name string, id string, ref *Relationship) error
}

// UnmarshalSelf extracts the data out of item, unmarshals it to out, then
// unmarshals the entire item to a [Relationship]. The item is assumed to
// be a self-relationship.
func UnmarshalSelf(item Item, out any) (Relationship, error) {
	var rel Relationship
	if err := attributevalue.UnmarshalMap(item, &rel); err != nil {
		return rel, fmt.Errorf("failed to unmarshal relationship: %w", err)
	}

	if data, ok := item[AttributeNameData]; !ok {
		return rel, fmt.Errorf("data attribute not found")
	} else if err := attributevalue.Unmarshal(data, &out); err != nil {
		return rel, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	unmarshaler, ok := out.(Unmarshaler)
	if !ok {
		return rel, nil
	}

	if err := unmarshaler.UnmarshalSelf(&rel); err != nil {
		return rel, fmt.Errorf("failed to unmarshal self: %w", err)
	}

	return rel, nil
}

// UnmarshalTableKey extracts and unmarshals the source and target keys from a DynamoDB item.
// Returns the source key, target key and any error that occurred during unmarshaling.
// Returns an error if either key is missing from the item.
func UnmarshalTableKey(item Item) (source, target string, err error) {
	var (
		hk, hkexists = item[AttributeNameSource]
		sk, skexists = item[AttributeNameTarget]
	)

	if !hkexists || !skexists {
		return "", "", fmt.Errorf("source and target keys not found")
	}

	err = errors.Join(
		attributevalue.Unmarshal(hk, &source),
		attributevalue.Unmarshal(sk, &target),
	)

	return source, target, err
}

// UnmarshalEntity unmarshals data to out from each item in items, where:
//   - self relationships are applied via [UnmarshalSelf], and
//   - other relationships are applied via [RefUnmarshaler.UnmarshalRef].
//
// This function is usually called to extract results from a QueryEntity.
func UnmarshalEntity(items []Item, out RefUnmarshaler, opts ...func(*MarshalOptions)) ([]Relationship, error) {
	if len(items) == 0 {
		return nil, ErrItemNotFound
	}

	var (
		marshalOpts   = newMarshalOptions(opts...)
		relationships []Relationship
	)

	for _, item := range items {
		source, target, err := UnmarshalTableKey(item)

		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal table key: %w", err)
		}

		// Check if this is a self relationship
		if source == target {
			if rel, err := UnmarshalSelf(item, &out); err != nil {
				return nil, fmt.Errorf("failed to unmarshal self: %w", err)
			} else {
				relationships = append(relationships, rel)
			}
		} else {
			data := Ref{}
			rel, err := UnmarshalSelf(item, &data)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal relationship: %w", err)
			}

			// Extract relationship name from label
			// Format: "<source_prefix>/<source_id>/<relationship_name>"
			_, id, name, err := marshalOpts.splitLabel(rel)
			if err != nil {
				return nil, fmt.Errorf("invalid label format: %s", rel.Label)
			}

			if err := out.UnmarshalRef(name, id, &rel); err != nil {
				return nil, fmt.Errorf("failed to unmarshal ref %s: %w", name, err)
			}

			relationships = append(relationships, rel)
		}
	}

	return relationships, nil
}

// UnmarshalList calls [UnmarshalSelf] on each item in items and stores the result in out.
// This function is usually called to extract results from [QueryList].
func UnmarshalList[T any](items []Item, out *[]T) ([]Relationship, error) {
	var relationships []Relationship

	for i, item := range items {
		var value T
		rel, err := UnmarshalSelf(item, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal item %d: %w", i, err)
		}
		*out = append(*out, value)
		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// DynamoDBClient interface for easier testing and connection management.
type DynamoDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}
