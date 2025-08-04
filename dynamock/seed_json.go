package dynamock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/nisimpson/dynamap"
)

// JSONAPIDocument represents the root structure of a JSON:API document.
// It can contain either a single resource or an array of resources.
type JSONAPIDocument []JSONAPIResource

// JSONAPIResource represents a single resource in JSON:API format.
type JSONAPIResource struct {
	Type          string                         `json:"type"`
	ID            string                         `json:"id"`
	Attributes    map[string]interface{}         `json:"attributes,omitempty"`
	Relationships map[string]JSONAPIRelationship `json:"relationships,omitempty"`
}

// JSONAPIRelationship represents a relationship in JSON:API format.
type JSONAPIRelationship struct {
	Data interface{} `json:"data"` // Can be JSONAPIResourceIdentifier, []JSONAPIResourceIdentifier, or nil
}

// JSONAPIResourceIdentifier represents a resource identifier in JSON:API format.
type JSONAPIResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// SeedFromJSON converts test data from a JSON:API formatted reader into test entities
// and persists them to the database. The JSON document format is expected to adhere
// to the JSON:API specification, as an array of primary documents.
// Returns the number of items saved and any errors generated.
func (s *SeedTestData) SeedFromJSON(ctx context.Context, r io.Reader) (int, error) {
	// Parse JSON document
	var document JSONAPIDocument
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&document); err != nil {
		return 0, fmt.Errorf("failed to parse JSON document: %w", err)
	}

	// Convert JSON:API resources to TestEntity instances
	entities := make([]*TestEntity, 0, len(document))
	for i, resource := range document {
		entity, err := s.convertResourceToEntity(resource)
		if err != nil {
			return 0, fmt.Errorf("failed to convert resource at index %d: %w", i, err)
		}
		entities = append(entities, entity)
	}

	// Seed entities to database
	count := 0
	for _, entity := range entities {
		if err := s.SeedEntity(ctx, entity); err != nil {
			return count, fmt.Errorf("failed to seed entity %s#%s: %w", entity.opts.SourcePrefix, entity.opts.SourceID, err)
		}
		count++
	}

	return count, nil
}

// convertResourceToEntity converts a JSON:API resource to a TestEntity.
func (s *SeedTestData) convertResourceToEntity(resource JSONAPIResource) (*TestEntity, error) {
	// Validate required fields
	if resource.Type == "" {
		return nil, fmt.Errorf("resource missing required 'type' field")
	}
	if resource.ID == "" {
		return nil, fmt.Errorf("resource missing required 'id' field")
	}

	// Create base entity
	entity := NewEntity(
		WithID(resource.ID),
		WithPrefix(resource.Type),
		WithLabel(resource.Type),
		WithData(resource.Attributes),
	).Build()

	// Process relationships
	if resource.Relationships != nil {
		for relationshipName, relationship := range resource.Relationships {
			relatedEntities, err := s.convertRelationshipData(relationship.Data, resource.Type, resource.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to convert relationship '%s': %w", relationshipName, err)
			}

			// Add relationships to entity
			for _, relatedEntity := range relatedEntities {
				entity.relationships[relationshipName] = append(entity.relationships[relationshipName], relatedEntity)
			}
		}
	}

	return entity, nil
}

// convertRelationshipData converts JSON:API relationship data to TestEntity instances.
func (s *SeedTestData) convertRelationshipData(data interface{}, sourceType, sourceID string) ([]dynamap.Marshaler, error) {
	if data == nil {
		return nil, nil
	}

	var identifiers []JSONAPIResourceIdentifier

	// Handle both single resource identifier and array of identifiers
	switch v := data.(type) {
	case map[string]interface{}:
		// Single resource identifier
		var identifier JSONAPIResourceIdentifier
		if err := s.mapToStruct(v, &identifier); err != nil {
			return nil, fmt.Errorf("failed to parse resource identifier: %w", err)
		}
		identifiers = []JSONAPIResourceIdentifier{identifier}
	case []interface{}:
		// Array of resource identifiers
		for i, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("relationship data item at index %d is not an object", i)
			}
			var identifier JSONAPIResourceIdentifier
			if err := s.mapToStruct(itemMap, &identifier); err != nil {
				return nil, fmt.Errorf("failed to parse resource identifier at index %d: %w", i, err)
			}
			identifiers = append(identifiers, identifier)
		}
	default:
		return nil, fmt.Errorf("relationship data must be an object or array of objects")
	}

	// Convert identifiers to TestEntity instances
	entities := make([]dynamap.Marshaler, 0, len(identifiers))
	for _, identifier := range identifiers {
		if identifier.Type == "" {
			return nil, fmt.Errorf("resource identifier missing required 'type' field")
		}
		if identifier.ID == "" {
			return nil, fmt.Errorf("resource identifier missing required 'id' field")
		}

		// Create entity for the relationship target
		entity := NewEntity(
			WithID(identifier.ID),
			WithPrefix(identifier.Type),
			WithLabel(identifier.Type),
		).Build()

		entities = append(entities, entity)
	}

	return entities, nil
}

// mapToStruct converts a map[string]interface{} to a struct using JSON marshaling/unmarshaling.
func (s *SeedTestData) mapToStruct(m map[string]interface{}, target interface{}) error {
	// Convert map to JSON and then unmarshal to struct
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
	}

	return nil
}
