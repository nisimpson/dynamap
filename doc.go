// Package dynamap provides a lightweight entity-relationship abstraction layer
// over the AWS SDK for Go v2 DynamoDB client.
//
// The library allows you to model relationships between domain objects and
// perform standard CRUD operations on a DynamoDB table with a consistent schema.
//
// # Key Concepts
//
// Entities implement the Marshaler interface to define how they map to DynamoDB
// relationships. RefMarshalers can also define relationships to other entities.
//
// The library uses a single-table design with the following schema:
//   - hk (hash key): source entity key (prefix#id)
//   - sk (sort key): target entity key (prefix#id)
//   - label: relationship type/category
//   - gsi1_sk: sort key for the ref index
//
// # Basic Usage
//
//	// Define your entity
//	type Product struct {
//	    ID       string `json:"id"`
//	    Category string `json:"category"`
//	}
//
//	func (p *Product) MarshalSelf(opts *dynamap.MarshalOptions) error {
//	    opts.SourcePrefix = "product"
//	    opts.SourceID = p.ID
//	    opts.TargetPrefix = "product"
//	    opts.TargetID = p.ID
//	    opts.Label = "product"
//	    opts.RefSortKey = p.Category
//	    return nil
//	}
//
//	// Use the library
//	table := dynamap.NewTable("my-table")
//	putInput, err := table.MarshalPut(product)
//	_, err = ddb.PutItem(ctx, putInput)
//
// # Relationships
//
// Entities can define relationships to other entities by implementing RefMarshaler:
//
//	func (o *Order) MarshalRefs(ctx *dynamap.RelationshipContext) error {
//	    ctx.AddMany("products", dynamap.SliceOf(o.Products...))
//	    return nil
//	}
//
// # Querying
//
// The library provides two main query types:
//   - QueryList: Query entities by label across the table
//   - QueryEntity: Query an entity and its relationships
//
// # Pagination
//
// Built-in pagination support stores cursors in the same table:
//
//	paginator := table.Paginator(ddb)
//	cursor, err := paginator.PageCursor(ctx, lastEvaluatedKey)
//	startKey, err := paginator.StartKey(ctx, cursor)
package dynamap

// This file serves as the main entry point for the dynamap package.
// All core functionality is implemented in the other files in this package.
