package dynamock

import (
	"testing"
	"time"

	"github.com/nisimpson/dynamap"
)

func TestNewEntity(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	entity := NewEntity(
		WithID("test-id"),
		WithPrefix("test"),
		WithLabel("test-label"),
		WithRefSortKey("sort-key"),
		WithCreated(fixedTime),
		WithUpdated(fixedTime),
		WithTimeToLive(time.Hour),
		WithKeyDelimiter("|"),
		WithLabelDelimiter("-"),
		WithData("test-data"),
	).Build()

	if entity.opts.SourceID != "test-id" {
		t.Errorf("expected source ID 'test-id', got %s", entity.opts.SourceID)
	}

	if entity.opts.TargetID != "test-id" {
		t.Errorf("expected target ID 'test-id', got %s", entity.opts.TargetID)
	}

	if entity.opts.SourcePrefix != "test" {
		t.Errorf("expected source prefix 'test', got %s", entity.opts.SourcePrefix)
	}

	if entity.opts.TargetPrefix != "test" {
		t.Errorf("expected target prefix 'test', got %s", entity.opts.TargetPrefix)
	}

	if entity.opts.Label != "test-label" {
		t.Errorf("expected label 'test-label', got %s", entity.opts.Label)
	}

	if entity.opts.RefSortKey != "sort-key" {
		t.Errorf("expected ref sort key 'sort-key', got %s", entity.opts.RefSortKey)
	}

	if !entity.opts.Created.Equal(fixedTime) {
		t.Errorf("expected created time %v, got %v", fixedTime, entity.opts.Created)
	}

	if !entity.opts.Updated.Equal(fixedTime) {
		t.Errorf("expected updated time %v, got %v", fixedTime, entity.opts.Updated)
	}

	if entity.opts.TimeToLive != time.Hour {
		t.Errorf("expected TTL 1h, got %v", entity.opts.TimeToLive)
	}

	if entity.opts.KeyDelimiter != "|" {
		t.Errorf("expected key delimiter '|', got %s", entity.opts.KeyDelimiter)
	}

	if entity.opts.LabelDelimiter != "-" {
		t.Errorf("expected label delimiter '-', got %s", entity.opts.LabelDelimiter)
	}

	if entity.data != "test-data" {
		t.Errorf("expected data 'test-data', got %v", entity.data)
	}
}

func TestFunctionalOptions_SeparateSourceTarget(t *testing.T) {
	entity := NewEntity(
		WithSourceID("source-id"),
		WithTargetID("target-id"),
		WithSourcePrefix("source"),
		WithTargetPrefix("target"),
	).Build()

	if entity.opts.SourceID != "source-id" {
		t.Errorf("expected source ID 'source-id', got %s", entity.opts.SourceID)
	}

	if entity.opts.TargetID != "target-id" {
		t.Errorf("expected target ID 'target-id', got %s", entity.opts.TargetID)
	}

	if entity.opts.SourcePrefix != "source" {
		t.Errorf("expected source prefix 'source', got %s", entity.opts.SourcePrefix)
	}

	if entity.opts.TargetPrefix != "target" {
		t.Errorf("expected target prefix 'target', got %s", entity.opts.TargetPrefix)
	}
}

func TestFunctionalOptions_WithRelationships(t *testing.T) {
	relatedEntity1 := NewEntity(WithID("P1"), WithPrefix("product"), WithLabel("product")).Build()
	relatedEntity2 := NewEntity(WithID("P2"), WithPrefix("product"), WithLabel("product")).Build()
	relatedEntity3 := NewEntity(WithID("P3"), WithPrefix("product"), WithLabel("product")).Build()

	entity := NewEntity(
		WithID("test-id"),
		WithRelationship("products", relatedEntity1),
		WithRelationship("products", relatedEntity2),
		WithRelationships("categories", relatedEntity3),
	).Build()

	if len(entity.relationships["products"]) != 2 {
		t.Errorf("expected 2 product relationships, got %d", len(entity.relationships["products"]))
	}

	if len(entity.relationships["categories"]) != 1 {
		t.Errorf("expected 1 category relationship, got %d", len(entity.relationships["categories"]))
	}
}

func TestTestEntity_MarshalSelf(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	entity := NewEntity(
		WithID("test-id"),
		WithPrefix("test"),
		WithLabel("test-label"),
		WithRefSortKey("sort-key"),
		WithCreated(fixedTime),
		WithUpdated(fixedTime),
		WithTimeToLive(time.Hour),
		WithKeyDelimiter("|"),
		WithLabelDelimiter("-"),
	).Build()

	opts := &dynamap.MarshalOptions{}
	err := entity.MarshalSelf(opts)
	if err != nil {
		t.Fatalf("MarshalSelf failed: %v", err)
	}

	if opts.SourceID != "test-id" {
		t.Errorf("expected source ID 'test-id', got %s", opts.SourceID)
	}

	if opts.SourcePrefix != "test" {
		t.Errorf("expected source prefix 'test', got %s", opts.SourcePrefix)
	}

	if opts.Label != "test-label" {
		t.Errorf("expected label 'test-label', got %s", opts.Label)
	}

	if opts.RefSortKey != "sort-key" {
		t.Errorf("expected ref sort key 'sort-key', got %s", opts.RefSortKey)
	}

	if !opts.Created.Equal(fixedTime) {
		t.Errorf("expected created time %v, got %v", fixedTime, opts.Created)
	}

	if opts.TimeToLive != time.Hour {
		t.Errorf("expected TTL 1h, got %v", opts.TimeToLive)
	}

	if opts.KeyDelimiter != "|" {
		t.Errorf("expected key delimiter '|', got %s", opts.KeyDelimiter)
	}

	if opts.LabelDelimiter != "-" {
		t.Errorf("expected label delimiter '-', got %s", opts.LabelDelimiter)
	}
}

func TestTestEntity_MarshalRefs(t *testing.T) {
	relatedEntity1 := NewEntity(WithID("P1"), WithPrefix("product"), WithLabel("product")).Build()
	relatedEntity2 := NewEntity(WithID("P2"), WithPrefix("product"), WithLabel("product")).Build()

	entity := NewEntity(
		WithID("test-id"),
		WithRelationship("products", relatedEntity1),
		WithRelationship("products", relatedEntity2),
	).Build()

	// Test that MarshalRefs works with dynamap
	relationships, err := dynamap.MarshalRelationships(entity)
	if err != nil {
		t.Fatalf("MarshalRelationships failed: %v", err)
	}

	// Should have at least the self relationship plus the product relationships
	if len(relationships) < 3 {
		t.Errorf("expected at least 3 relationships (1 self + 2 products), got %d", len(relationships))
	}
}

func TestTestEntity_UnmarshalSelf(t *testing.T) {
	// Create a relationship to unmarshal from
	createdTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)

	rel := &dynamap.Relationship{
		CreatedAt: createdTime,
		UpdatedAt: updatedTime,
		Data:      "test-data",
	}

	entity := NewEntity().Build()
	err := entity.UnmarshalSelf(rel)
	if err != nil {
		t.Fatalf("UnmarshalSelf failed: %v", err)
	}

	if !entity.opts.Created.Equal(createdTime) {
		t.Errorf("expected created time %v, got %v", createdTime, entity.opts.Created)
	}

	if !entity.opts.Updated.Equal(updatedTime) {
		t.Errorf("expected updated time %v, got %v", updatedTime, entity.opts.Updated)
	}

	if entity.data != "test-data" {
		t.Errorf("expected data 'test-data', got %v", entity.data)
	}
}

func TestTestEntity_UnmarshalRef(t *testing.T) {
	entity := NewEntity(WithID("test-id")).Build()

	// Create a relationship to unmarshal
	rel := &dynamap.Relationship{
		Data: "related-data",
	}

	err := entity.UnmarshalRef("products", "P1", rel)
	if err != nil {
		t.Fatalf("UnmarshalRef failed: %v", err)
	}

	if len(entity.relationships["products"]) != 1 {
		t.Errorf("expected 1 product relationship, got %d", len(entity.relationships["products"]))
	}

	// Test adding another relationship
	err = entity.UnmarshalRef("products", "P2", rel)
	if err != nil {
		t.Fatalf("UnmarshalRef failed: %v", err)
	}

	if len(entity.relationships["products"]) != 2 {
		t.Errorf("expected 2 product relationships, got %d", len(entity.relationships["products"]))
	}
}

func TestTestEntity_InterfaceCompliance(t *testing.T) {
	entity := NewEntity().Build()

	// Test that TestEntity implements all required interfaces
	var _ dynamap.Marshaler = entity
	var _ dynamap.RefMarshaler = entity
	var _ dynamap.Unmarshaler = entity
	var _ dynamap.RefUnmarshaler = entity
}

func TestTestEntity_FullWorkflow(t *testing.T) {
	// Test a complete marshal/unmarshal workflow
	originalEntity := NewEntity(
		WithID("E1"),
		WithPrefix("entity"),
		WithLabel("entity"),
		WithRefSortKey("test"),
		WithData(map[string]interface{}{"field": "value"}),
	).Build()

	// Add a relationship
	relatedEntity := NewEntity(
		WithID("R1"),
		WithPrefix("related"),
		WithLabel("related"),
	).Build()

	originalEntity.relationships = map[string][]dynamap.Marshaler{
		"related": {relatedEntity},
	}

	// Marshal the entity
	relationships, err := dynamap.MarshalRelationships(originalEntity)
	if err != nil {
		t.Fatalf("MarshalRelationships failed: %v", err)
	}

	if len(relationships) != 2 {
		t.Errorf("expected 2 relationships (1 self + 1 related), got %d", len(relationships))
	}

	// Create a new entity and unmarshal into it
	newEntity := NewEntity().Build()

	// Find the self relationship and unmarshal it
	for _, rel := range relationships {
		if rel.Source == "entity#E1" && rel.Target == "entity#E1" {
			err = newEntity.UnmarshalSelf(&rel)
			if err != nil {
				t.Fatalf("UnmarshalSelf failed: %v", err)
			}
		} else {
			// This is a related entity
			err = newEntity.UnmarshalRef("related", "R1", &rel)
			if err != nil {
				t.Fatalf("UnmarshalRef failed: %v", err)
			}
		}
	}

	// Verify the unmarshaled entity has the relationship
	if len(newEntity.relationships["related"]) != 1 {
		t.Errorf("expected 1 related relationship after unmarshal, got %d", len(newEntity.relationships["related"]))
	}
}
