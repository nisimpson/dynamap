package dynamock

import (
	"time"

	"github.com/nisimpson/dynamap"
)

// EntityOption is a functional option for configuring entities during building.
type EntityOption func(*EntityBuilder)

// EntityBuilder provides entity building through functional options only.
type EntityBuilder struct {
	*TestEntity
}

// NewEntity creates a new entity builder with the given options applied.
func NewEntity(opts ...EntityOption) *EntityBuilder {
	builder := &EntityBuilder{
		TestEntity: &TestEntity{
			opts:          dynamap.NewMarshalOptions(),
			relationships: make(map[string][]dynamap.Marshaler),
		},
	}
	for _, opt := range opts {
		opt(builder)
	}
	return builder
}

// Build creates a TestEntity from the builder configuration.
func (b *EntityBuilder) Build() *TestEntity {
	return &TestEntity{
		opts:          b.opts,
		data:          b.data,
		relationships: b.relationships,
	}
}

// Functional Options

// WithID sets both source and target ID to the same value (for self-referencing entities).
func WithID(id string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.SourceID = id
		b.opts.TargetID = id
	}
}

// WithSourceID sets the source ID.
func WithSourceID(id string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.SourceID = id
	}
}

// WithTargetID sets the target ID.
func WithTargetID(id string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.TargetID = id
	}
}

// WithPrefix sets both source and target prefix to the same value (for self-referencing entities).
func WithPrefix(prefix string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.SourcePrefix = prefix
		b.opts.TargetPrefix = prefix
	}
}

// WithSourcePrefix sets the source prefix.
func WithSourcePrefix(prefix string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.SourcePrefix = prefix
	}
}

// WithTargetPrefix sets the target prefix.
func WithTargetPrefix(prefix string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.TargetPrefix = prefix
	}
}

// WithLabel sets the relationship label.
func WithLabel(label string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.Label = label
	}
}

// WithRefSortKey sets the reference sort key.
func WithRefSortKey(sortKey string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.RefSortKey = sortKey
	}
}

// WithCreated sets the creation timestamp.
func WithCreated(created time.Time) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.Created = created
	}
}

// WithUpdated sets the update timestamp.
func WithUpdated(updated time.Time) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.Updated = updated
	}
}

// WithTimeToLive sets the TTL duration.
func WithTimeToLive(ttl time.Duration) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.TimeToLive = ttl
	}
}

// WithKeyDelimiter sets the key delimiter.
func WithKeyDelimiter(delimiter string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.KeyDelimiter = delimiter
	}
}

// WithLabelDelimiter sets the label delimiter.
func WithLabelDelimiter(delimiter string) EntityOption {
	return func(b *EntityBuilder) {
		b.opts.LabelDelimiter = delimiter
	}
}

// WithData sets the entity data.
func WithData(data interface{}) EntityOption {
	return func(b *EntityBuilder) {
		b.data = data
	}
}

// WithRelationship adds a single relationship.
func WithRelationship(name string, entity dynamap.Marshaler) EntityOption {
	return func(b *EntityBuilder) {
		if b.relationships[name] == nil {
			b.relationships[name] = make([]dynamap.Marshaler, 0)
		}
		b.relationships[name] = append(b.relationships[name], entity)
	}
}

// WithRelationships adds multiple relationships with the same name.
func WithRelationships(name string, entities ...dynamap.Marshaler) EntityOption {
	return func(b *EntityBuilder) {
		if b.relationships[name] == nil {
			b.relationships[name] = make([]dynamap.Marshaler, 0)
		}
		b.relationships[name] = append(b.relationships[name], entities...)
	}
}

// TestEntity is a generic test entity that implements all dynamap interfaces.
type TestEntity struct {
	opts          dynamap.MarshalOptions
	data          interface{}
	relationships map[string][]dynamap.Marshaler
}

// MarshalSelf implements the dynamap.Marshaler interface.
func (e *TestEntity) MarshalSelf(opts *dynamap.MarshalOptions) error {
	opts.SourcePrefix = e.opts.SourcePrefix
	opts.SourceID = e.opts.SourceID
	opts.TargetPrefix = e.opts.TargetPrefix
	opts.TargetID = e.opts.TargetID
	opts.Label = e.opts.Label
	opts.RefSortKey = e.opts.RefSortKey
	opts.Created = e.opts.Created
	opts.Updated = e.opts.Updated
	opts.TimeToLive = e.opts.TimeToLive
	opts.KeyDelimiter = e.opts.KeyDelimiter
	opts.LabelDelimiter = e.opts.LabelDelimiter
	return nil
}

// MarshalRefs implements the dynamap.RefMarshaler interface.
func (e *TestEntity) MarshalRefs(ctx *dynamap.RelationshipContext) error {
	for name, entities := range e.relationships {
		ctx.AddMany(name, entities)
	}
	return nil
}

// UnmarshalSelf implements the dynamap.Unmarshaler interface.
func (e *TestEntity) UnmarshalSelf(rel *dynamap.Relationship) error {
	// Extract basic information from the relationship
	e.opts.Created = rel.CreatedAt
	e.opts.Updated = rel.UpdatedAt

	// If there's data in the relationship, store it
	if rel.Data != nil {
		e.data = rel.Data
	}

	return nil
}

// UnmarshalRef implements the dynamap.RefUnmarshaler interface.
func (e *TestEntity) UnmarshalRef(name string, id string, ref *dynamap.Relationship) error {
	// For TestEntity, we'll create a simple entity for each relationship
	// Users can override this behavior in their own entities
	relatedEntity := NewEntity(
		WithID(id),
		WithData(ref.Data),
	).Build()

	if e.relationships == nil {
		e.relationships = make(map[string][]dynamap.Marshaler)
	}

	if e.relationships[name] == nil {
		e.relationships[name] = make([]dynamap.Marshaler, 0)
	}

	e.relationships[name] = append(e.relationships[name], relatedEntity)
	return nil
}

// Ensure TestEntity implements all required interfaces
var _ dynamap.Marshaler = (*TestEntity)(nil)
var _ dynamap.RefMarshaler = (*TestEntity)(nil)
var _ dynamap.Unmarshaler = (*TestEntity)(nil)
var _ dynamap.RefUnmarshaler = (*TestEntity)(nil)
