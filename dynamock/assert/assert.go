// Package assert provides fluent assertion utilities for testing DynamoDB operations
// and dynamap entities. It makes tests more readable and maintainable by providing
// expressive assertion methods.
//
// # Usage
//
//	import "github.com/nisimpson/dynamap/dynamock/assert"
//
//	// Assert on DynamoDB items
//	assert.Items(t, result.Items).
//		HasCount(3).
//		ContainsEntity("product", "P1").
//		HasAttribute("label", "product")
//
//	// Assert on relationships
//	assert.Relationships(t, relationships).
//		HasSelfRelationship("order", "O1").
//		HasRelationship("order", "O1", "product", "P1")
//
//	// Assert on entities
//	assert.Entity(t, entity).
//		CanMarshal().
//		HasSourceID("E1").
//		HasLabel("entity")
package assert

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/nisimpson/dynamap"
	"github.com/nisimpson/dynamap/dynamock"
)

// ItemsAssertion provides fluent assertions for DynamoDB items.
type ItemsAssertion struct {
	t     *testing.T
	items []map[string]types.AttributeValue
}

// Items creates a new ItemsAssertion for the given DynamoDB items.
func Items(t *testing.T, items []map[string]types.AttributeValue) *ItemsAssertion {
	return &ItemsAssertion{
		t:     t,
		items: items,
	}
}

// HasCount asserts that the items collection has the expected count.
func (a *ItemsAssertion) HasCount(expected int) *ItemsAssertion {
	if len(a.items) != expected {
		a.t.Errorf("expected %d items, got %d", expected, len(a.items))
	}
	return a
}

// IsEmpty asserts that the items collection is empty.
func (a *ItemsAssertion) IsEmpty() *ItemsAssertion {
	return a.HasCount(0)
}

// IsNotEmpty asserts that the items collection is not empty.
func (a *ItemsAssertion) IsNotEmpty() *ItemsAssertion {
	if len(a.items) == 0 {
		a.t.Error("expected items to not be empty")
	}
	return a
}

// ContainsEntity asserts that the items contain an entity with the given prefix and ID.
func (a *ItemsAssertion) ContainsEntity(prefix, id string) *ItemsAssertion {
	expectedKey := fmt.Sprintf("%s#%s", prefix, id)

	for _, item := range a.items {
		hkStr, _ := a.getItemKeys(item)
		if hkStr == expectedKey {
			return a // Found the entity
		}
	}

	a.t.Errorf("expected to find entity %s#%s in items", prefix, id)
	return a
}

// ContainsRelationship asserts that the items contain a relationship between source and target.
func (a *ItemsAssertion) ContainsRelationship(sourcePrefix, sourceID, targetPrefix, targetID string) *ItemsAssertion {
	expectedSource := fmt.Sprintf("%s#%s", sourcePrefix, sourceID)
	expectedTarget := fmt.Sprintf("%s#%s", targetPrefix, targetID)

	for _, item := range a.items {
		hkStr, skStr := a.getItemKeys(item)
		if hkStr == expectedSource && skStr == expectedTarget {
			return a // Found the relationship
		}
	}

	a.t.Errorf("expected to find relationship from %s to %s in items", expectedSource, expectedTarget)
	return a
}

// HasAttribute asserts that at least one item has the specified attribute with the expected value.
func (a *ItemsAssertion) HasAttribute(attributeName, expectedValue string) *ItemsAssertion {
	for _, item := range a.items {
		if a.itemHasAttributeValue(item, attributeName, expectedValue) {
			return a // Found the attribute
		}
	}

	a.t.Errorf("expected to find attribute %s with value %s in items", attributeName, expectedValue)
	return a
}

// itemHasAttributeValue checks if an item has the specified attribute with the expected value.
func (a *ItemsAssertion) itemHasAttributeValue(item map[string]types.AttributeValue, attributeName, expectedValue string) bool {
	attr, exists := item[attributeName]
	if !exists {
		return false
	}

	attrStr, ok := attr.(*types.AttributeValueMemberS)
	if !ok {
		return false
	}

	return attrStr.Value == expectedValue
}

// ContainsEntityWithLabel asserts that the items contain an entity with the given label.
func (a *ItemsAssertion) ContainsEntityWithLabel(label string) *ItemsAssertion {
	for _, item := range a.items {
		if a.itemHasLabel(item, label) && a.itemIsEntity(item) {
			return a // Found entity with label
		}
	}

	a.t.Errorf("expected to find entity with label %s in items", label)
	return a
}

// itemHasLabel checks if an item has the exact label.
func (a *ItemsAssertion) itemHasLabel(item map[string]types.AttributeValue, label string) bool {
	labelAttr, exists := item["label"]
	if !exists {
		return false
	}

	labelStr, ok := labelAttr.(*types.AttributeValueMemberS)
	if !ok {
		return false
	}

	return labelStr.Value == label
}

// itemIsEntity checks if an item represents an entity (hk == sk).
func (a *ItemsAssertion) itemIsEntity(item map[string]types.AttributeValue) bool {
	hkStr, skStr := a.getItemKeys(item)
	if hkStr == "" || skStr == "" {
		return false
	}

	return hkStr == skStr
}

// ContainsRelationshipWithLabel asserts that the items contain a relationship with the given label.
func (a *ItemsAssertion) ContainsRelationshipWithLabel(label string) *ItemsAssertion {
	for _, item := range a.items {
		if a.itemHasLabelContaining(item, label) && a.itemIsRelationship(item) {
			return a // Found relationship with label
		}
	}

	a.t.Errorf("expected to find relationship with label containing %s in items", label)
	return a
}

// itemHasLabelContaining checks if an item has a label attribute containing the given text.
func (a *ItemsAssertion) itemHasLabelContaining(item map[string]types.AttributeValue, label string) bool {
	labelAttr, exists := item["label"]
	if !exists {
		return false
	}

	labelStr, ok := labelAttr.(*types.AttributeValueMemberS)
	if !ok {
		return false
	}

	return strings.Contains(labelStr.Value, label)
}

// itemIsRelationship checks if an item represents a relationship (hk != sk).
func (a *ItemsAssertion) itemIsRelationship(item map[string]types.AttributeValue) bool {
	hkStr, skStr := a.getItemKeys(item)
	if hkStr == "" || skStr == "" {
		return false
	}

	return hkStr != skStr
}

// getItemKeys extracts the hk and sk string values from an item.
func (a *ItemsAssertion) getItemKeys(item map[string]types.AttributeValue) (hk, sk string) {
	if hkAttr, exists := item["hk"]; exists {
		if hkStr, ok := hkAttr.(*types.AttributeValueMemberS); ok {
			hk = hkStr.Value
		}
	}

	if skAttr, exists := item["sk"]; exists {
		if skStr, ok := skAttr.(*types.AttributeValueMemberS); ok {
			sk = skStr.Value
		}
	}

	return hk, sk
}

// RelationshipsAssertion provides fluent assertions for dynamap relationships.
type RelationshipsAssertion struct {
	t             *testing.T
	relationships []dynamap.Relationship
}

// Relationships creates a new RelationshipsAssertion for the given relationships.
func Relationships(t *testing.T, relationships []dynamap.Relationship) *RelationshipsAssertion {
	return &RelationshipsAssertion{
		t:             t,
		relationships: relationships,
	}
}

// HasCount asserts that the relationships collection has the expected count.
func (a *RelationshipsAssertion) HasCount(expected int) *RelationshipsAssertion {
	if len(a.relationships) != expected {
		a.t.Errorf("expected %d relationships, got %d", expected, len(a.relationships))
	}
	return a
}

// HasSelfRelationship asserts that there is a self-relationship for the given entity.
func (a *RelationshipsAssertion) HasSelfRelationship(prefix, id string) *RelationshipsAssertion {
	expectedKey := fmt.Sprintf("%s#%s", prefix, id)

	for _, rel := range a.relationships {
		if rel.Source == expectedKey && rel.Target == expectedKey {
			return a // Found the self-relationship
		}
	}

	a.t.Errorf("expected to find self-relationship for %s#%s", prefix, id)
	return a
}

// HasRelationship asserts that there is a relationship from source to target.
func (a *RelationshipsAssertion) HasRelationship(sourcePrefix, sourceID, targetPrefix, targetID string) *RelationshipsAssertion {
	expectedSource := fmt.Sprintf("%s#%s", sourcePrefix, sourceID)
	expectedTarget := fmt.Sprintf("%s#%s", targetPrefix, targetID)

	for _, rel := range a.relationships {
		if rel.Source == expectedSource && rel.Target == expectedTarget {
			return a // Found the relationship
		}
	}

	a.t.Errorf("expected to find relationship from %s to %s", expectedSource, expectedTarget)
	return a
}

// HasLabel asserts that at least one relationship has the specified label.
func (a *RelationshipsAssertion) HasLabel(expectedLabel string) *RelationshipsAssertion {
	for _, rel := range a.relationships {
		if rel.Label == expectedLabel {
			return a // Found the label
		}
	}

	a.t.Errorf("expected to find relationship with label %s", expectedLabel)
	return a
}

// EntityAssertion provides fluent assertions for TestEntity instances.
type EntityAssertion struct {
	t      *testing.T
	entity *dynamock.TestEntity
}

// Entity creates a new EntityAssertion for the given TestEntity.
func Entity(t *testing.T, entity *dynamock.TestEntity) *EntityAssertion {
	return &EntityAssertion{
		t:      t,
		entity: entity,
	}
}

// CanMarshal asserts that the entity can marshal itself without error.
func (a *EntityAssertion) CanMarshal() *EntityAssertion {
	opts := &dynamap.MarshalOptions{}
	if err := a.entity.MarshalSelf(opts); err != nil {
		a.t.Errorf("entity failed to marshal: %v", err)
	}
	return a
}

// HasSourceID asserts that the entity marshals with the expected source ID.
func (a *EntityAssertion) HasSourceID(expectedID string) *EntityAssertion {
	opts := &dynamap.MarshalOptions{}
	if err := a.entity.MarshalSelf(opts); err != nil {
		a.t.Errorf("entity failed to marshal: %v", err)
		return a
	}

	if opts.SourceID != expectedID {
		a.t.Errorf("expected source ID %s, got %s", expectedID, opts.SourceID)
	}
	return a
}

// HasLabel asserts that the entity marshals with the expected label.
func (a *EntityAssertion) HasLabel(expectedLabel string) *EntityAssertion {
	opts := &dynamap.MarshalOptions{}
	if err := a.entity.MarshalSelf(opts); err != nil {
		a.t.Errorf("entity failed to marshal: %v", err)
		return a
	}

	if opts.Label != expectedLabel {
		a.t.Errorf("expected label %s, got %s", expectedLabel, opts.Label)
	}
	return a
}

// HasRefSortKey asserts that the entity marshals with the expected ref sort key.
func (a *EntityAssertion) HasRefSortKey(expectedSortKey string) *EntityAssertion {
	opts := &dynamap.MarshalOptions{}
	if err := a.entity.MarshalSelf(opts); err != nil {
		a.t.Errorf("entity failed to marshal: %v", err)
		return a
	}

	if opts.RefSortKey != expectedSortKey {
		a.t.Errorf("expected ref sort key %s, got %s", expectedSortKey, opts.RefSortKey)
	}
	return a
}

// CanMarshalRelationships asserts that the entity can marshal its relationships without error.
func (a *EntityAssertion) CanMarshalRelationships() *EntityAssertion {
	_, err := dynamap.MarshalRelationships(a.entity)
	if err != nil {
		a.t.Errorf("entity failed to marshal relationships: %v", err)
	}
	return a
}

// HasRelationshipCount asserts that the entity marshals the expected number of relationships.
func (a *EntityAssertion) HasRelationshipCount(expected int) *EntityAssertion {
	relationships, err := dynamap.MarshalRelationships(a.entity)
	if err != nil {
		a.t.Errorf("entity failed to marshal relationships: %v", err)
		return a
	}

	if len(relationships) != expected {
		a.t.Errorf("expected %d relationships, got %d", expected, len(relationships))
	}
	return a
}

// DynamoDBItemAssertion provides fluent assertions for individual DynamoDB items.
type DynamoDBItemAssertion struct {
	t    *testing.T
	item map[string]types.AttributeValue
}

// DynamoDBItem creates a new DynamoDBItemAssertion for the given item.
func DynamoDBItem(t *testing.T, item map[string]types.AttributeValue) *DynamoDBItemAssertion {
	return &DynamoDBItemAssertion{
		t:    t,
		item: item,
	}
}

// HasKey asserts that the item has the specified key with the expected value.
func (a *DynamoDBItemAssertion) HasKey(keyName, expectedValue string) *DynamoDBItemAssertion {
	if attr, exists := a.item[keyName]; !exists {
		a.t.Errorf("item missing key %s", keyName)
	} else if attrStr, ok := attr.(*types.AttributeValueMemberS); !ok {
		a.t.Errorf("key %s is not a string", keyName)
	} else if attrStr.Value != expectedValue {
		a.t.Errorf("key %s expected %s, got %s", keyName, expectedValue, attrStr.Value)
	}
	return a
}

// HasAttribute asserts that the item has the specified attribute with the expected value.
func (a *DynamoDBItemAssertion) HasAttribute(attrName, expectedValue string) *DynamoDBItemAssertion {
	return a.HasKey(attrName, expectedValue) // Same implementation for now
}

// HasDataField asserts that the item's data attribute contains the specified field.
func (a *DynamoDBItemAssertion) HasDataField(fieldName, expectedValue string) *DynamoDBItemAssertion {
	dataAttr, exists := a.item["data"]
	if !exists {
		a.t.Error("item missing data attribute")
		return a
	}

	dataMap, ok := dataAttr.(*types.AttributeValueMemberM)
	if !ok {
		a.t.Error("data attribute is not a map")
		return a
	}

	fieldAttr, exists := dataMap.Value[fieldName]
	if !exists {
		a.t.Errorf("data missing field %s", fieldName)
		return a
	}

	if fieldStr, ok := fieldAttr.(*types.AttributeValueMemberS); ok {
		if fieldStr.Value != expectedValue {
			a.t.Errorf("data field %s expected %s, got %s", fieldName, expectedValue, fieldStr.Value)
		}
	} else {
		a.t.Errorf("data field %s is not a string", fieldName)
	}

	return a
}

// IsEntity asserts that the item represents an entity (hk == sk).
func (a *DynamoDBItemAssertion) IsEntity() *DynamoDBItemAssertion {
	hkStr, skStr := a.getKeys()
	if hkStr == "" || skStr == "" {
		a.t.Error("item missing hk or sk attribute")
		return a
	}

	if hkStr != skStr {
		a.t.Errorf("expected entity (hk == sk), but hk=%s, sk=%s", hkStr, skStr)
	}

	return a
}

// IsRelationship asserts that the item represents a relationship (hk != sk).
func (a *DynamoDBItemAssertion) IsRelationship() *DynamoDBItemAssertion {
	hkStr, skStr := a.getKeys()
	if hkStr == "" || skStr == "" {
		a.t.Error("item missing hk or sk attribute")
		return a
	}

	if hkStr == skStr {
		a.t.Errorf("expected relationship (hk != sk), but both are %s", hkStr)
	}

	return a
}

// getKeys extracts the hk and sk string values from the item.
func (a *DynamoDBItemAssertion) getKeys() (hk, sk string) {
	if hkAttr, exists := a.item["hk"]; exists {
		if hkStr, ok := hkAttr.(*types.AttributeValueMemberS); ok {
			hk = hkStr.Value
		}
	}

	if skAttr, exists := a.item["sk"]; exists {
		if skStr, ok := skAttr.(*types.AttributeValueMemberS); ok {
			sk = skStr.Value
		}
	}

	return hk, sk
}
