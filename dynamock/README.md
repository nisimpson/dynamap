# Dynamock

Dynamock provides comprehensive testing utilities for the Dynamap library, including mocking capabilities, integration testing helpers, test data builders, and JSON seeding functionality.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Features](#features)
  - [Mock Client](#mock-client)
  - [Local DynamoDB Integration](#local-dynamodb-integration)
  - [Test Data Builders](#test-data-builders)
  - [JSON Seeding](#json-seeding)
  - [Assertions](#assertions)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Contributing](#contributing)

## Overview

Dynamock serves multiple purposes in testing Dynamap applications:

1. **Mocking**: Provides a mock DynamoDB client for unit testing without external dependencies
2. **Integration Testing**: Utilities for testing against real DynamoDB (local or AWS)
3. **Test Data Management**: Builders and seeding utilities for creating complex test scenarios
4. **Assertions**: Fluent assertion helpers for validating test results

## Quick Start

### Mock Testing

```go
import (
    "context"
    "testing"
    "github.com/nisimpson/dynamap"
    "github.com/nisimpson/dynamap/dynamock"
)

func TestOrderCreation(t *testing.T) {
    // Create mock client
    mock := dynamock.NewMockClient(t)

    // Set up expectations
    mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
        // Verify the item being stored
        return &dynamodb.PutItemOutput{}, nil
    }

    // Test your code
    table := dynamap.NewTable("test-table")
    entity := dynamock.NewEntity(
        dynamock.WithID("O1"),
        dynamock.WithPrefix("order"),
        dynamock.WithLabel("order"),
    ).Build()

    putInput, err := table.MarshalPut(entity)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    _, err = mock.PutItem(context.Background(), putInput)
    if err != nil {
        t.Fatalf("Failed to put item: %v", err)
    }
}
```

### Integration Testing

```go
func TestOrderCreationIntegration(t *testing.T) {
    // Use local DynamoDB for integration testing
    dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
        // Create isolated table for this test
        dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
            // Your test code here - table is automatically cleaned up
            seeder := dynamock.NewSeedTestData(local.Client, tableName)

            entity := dynamock.NewEntity(
                dynamock.WithID("O1"),
                dynamock.WithPrefix("order"),
            ).Build()

            err := seeder.SeedEntity(context.Background(), entity)
            if err != nil {
                t.Fatalf("Failed to seed entity: %v", err)
            }
        })
    })
}
```

### JSON Seeding

```go
func TestWithJSONData(t *testing.T) {
    dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
        dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
            seeder := dynamock.NewSeedTestData(local.Client, tableName)

            // Load test data from JSON file
            file, err := os.Open("testdata/orders.json")
            if err != nil {
                t.Fatalf("Failed to open test data: %v", err)
            }
            defer file.Close()

            // Seed data from JSON
            count, err := seeder.SeedFromJSON(context.Background(), file)
            if err != nil {
                t.Fatalf("Failed to seed data: %v", err)
            }

            t.Logf("Seeded %d entities from JSON", count)

            // Your test logic here
        })
    })
}
```

## Features

### Mock Client

The mock client provides a simple way to test DynamoDB operations without external dependencies.

#### Basic Usage

```go
mock := dynamock.NewMockClient(t)

// Set up expectations for specific operations
mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
    // Your validation logic here
    return &dynamodb.PutItemOutput{}, nil
}

mock.GetFunc = func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
    // Return mock data
    return &dynamodb.GetItemOutput{
        Item: map[string]types.AttributeValue{
            "hk": &types.AttributeValueMemberS{Value: "order#O1"},
            "sk": &types.AttributeValueMemberS{Value: "order#O1"},
        },
    }, nil
}
```

#### Supported Operations

- `PutItem`
- `GetItem`
- `Query`
- `BatchWriteItem`
- `DeleteItem`
- `UpdateItem`

#### Error Simulation

```go
mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
    return nil, &types.ConditionalCheckFailedException{
        Message: aws.String("Item already exists"),
    }
}
```

### Local DynamoDB Integration

Dynamock provides utilities for testing against DynamoDB Local, enabling full integration testing.

#### Setup

```go
// Manual setup
client := dynamock.NewLocalClient(8000)
local := &dynamock.LocalDynamoDB{
    Client: client,
    Port:   8000,
}

// Helper functions
dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
    // Test code here
})

dynamock.WithIsolatedTable(t, client, func(tableName string) {
    // Test code with isolated table
})
```

#### Table Management

```go
// Create test table with unique name
tableName := dynamock.NewTestTable("my-test")

// Table lifecycle management
manager := dynamock.NewTableManager(client)
err := manager.CreateTestTable(tableName)
defer manager.Cleanup() // Cleans up all created tables
```

#### Environment Detection

Dynamock automatically detects the testing environment:

- Skips integration tests when DynamoDB Local is not available
- Provides clear error messages for setup issues
- Supports both local development and CI/CD environments

### Test Data Builders

Create test entities using functional options pattern for maximum flexibility.

#### Basic Entity Building

```go
// Simple entity
entity := dynamock.NewEntity(
    dynamock.WithID("E1"),
    dynamock.WithPrefix("entity"),
    dynamock.WithLabel("test-entity"),
    dynamock.WithData(map[string]interface{}{
        "name": "Test Entity",
        "value": 42,
    }),
).Build()

// Entity with relationships
parent := dynamock.NewEntity(
    dynamock.WithID("P1"),
    dynamock.WithPrefix("parent"),
    dynamock.WithLabel("parent"),
    dynamock.WithRelationships("children", child1, child2),
).Build()
```

#### Available Options

- `WithID(id)` - Sets both source and target ID
- `WithSourceID(id)` / `WithTargetID(id)` - Sets individual IDs
- `WithPrefix(prefix)` - Sets both source and target prefix
- `WithSourcePrefix(prefix)` / `WithTargetPrefix(prefix)` - Sets individual prefixes
- `WithLabel(label)` - Sets the relationship label
- `WithRefSortKey(sortKey)` - Sets the reference sort key
- `WithData(data)` - Sets the entity data
- `WithRelationship(name, entity)` - Adds a single relationship
- `WithRelationships(name, ...entities)` - Adds multiple relationships
- `WithCreated(time)` / `WithUpdated(time)` - Sets timestamps
- `WithTimeToLive(duration)` - Sets TTL
- `WithKeyDelimiter(delimiter)` / `WithLabelDelimiter(delimiter)` - Sets delimiters

#### TestEntity Interface

All built entities implement the complete dynamap interface set:

```go
type TestEntity struct {
    // implements dynamap.Marshaler
    // implements dynamap.RefMarshaler
    // implements dynamap.Unmarshaler
    // implements dynamap.RefUnmarshaler
}
```

### JSON Seeding

Bulk create test entities from JSON files following the JSON:API specification.

#### JSON Format

```json
[
  {
    "type": "order",
    "id": "O1",
    "attributes": {
      "customer_id": "C1",
      "total": 1348,
      "status": "pending"
    },
    "relationships": {
      "products": {
        "data": [
          {
            "type": "product",
            "id": "P1"
          },
          {
            "type": "product",
            "id": "P2"
          }
        ]
      },
      "customer": {
        "data": {
          "type": "customer",
          "id": "C1"
        }
      }
    }
  }
]
```

#### Field Mapping

| JSON:API Field  | TestEntity Field                        | Description                             |
| --------------- | --------------------------------------- | --------------------------------------- |
| `type`          | `sourcePrefix`, `targetPrefix`, `label` | Entity type/category                    |
| `id`            | `sourceID`, `targetID`                  | Entity identifier                       |
| `attributes`    | `data`                                  | Entity data as `map[string]interface{}` |
| `relationships` | `relationships`                         | Map of relationship name to entities    |

#### Usage

```go
seeder := dynamock.NewSeedTestData(client, tableName)

// From file
file, err := os.Open("testdata/orders.json")
if err != nil {
    t.Fatalf("Failed to open test data: %v", err)
}
defer file.Close()

count, err := seeder.SeedFromJSON(context.Background(), file)
if err != nil {
    t.Fatalf("Failed to seed data: %v", err)
}

// From string
jsonData := `[{"type": "product", "id": "P1", "attributes": {"name": "Laptop"}}]`
count, err = seeder.SeedFromJSON(context.Background(), strings.NewReader(jsonData))
```

#### Relationship Types

**Array Relationships (One-to-Many)**

```json
"products": {
  "data": [
    {"type": "product", "id": "P1"},
    {"type": "product", "id": "P2"}
  ]
}
```

**Single Relationships (One-to-One)**

```json
"customer": {
  "data": {
    "type": "customer",
    "id": "C1"
  }
}
```

**Null Relationships**

```json
"optional_field": {
  "data": null
}
```

### Assertions

The `dynamock/assert` subpackage provides fluent assertion utilities. See the [assert package README](./assert/README.md) for detailed documentation.

```go
import "github.com/nisimpson/dynamap/dynamock/assert"

// Assert on DynamoDB items
assert.Items(t, queryResult).
    HasCount(3).
    ContainsEntity("order", "O1").
    ContainsRelationship("order", "O1", "product", "P1")

// Assert on relationships
assert.Relationships(t, relationships).
    HasCount(2).
    HasSelfRelationship("order", "O1")

// Assert on individual items
assert.DynamoDBItem(t, item).
    IsEntity().
    HasKey("hk", "order#O1").
    HasDataField("customer_id", "C1")
```

## API Reference

### Core Types

#### MockClient

```go
type MockClient struct {
    PutFunc            DynamoDBAPICall[dynamodb.PutItemInput, dynamodb.PutItemOutput]
    GetFunc            DynamoDBAPICall[dynamodb.GetItemInput, dynamodb.GetItemOutput]
    QueryFunc          DynamoDBAPICall[dynamodb.QueryInput, dynamodb.QueryOutput]
    BatchWriteItemFunc DynamoDBAPICall[dynamodb.BatchWriteItemInput, dynamodb.BatchWriteItemOutput]
    DeleteItemFunc     DynamoDBAPICall[dynamodb.DeleteItemInput, dynamodb.DeleteItemOutput]
    UpdateItemFunc     DynamoDBAPICall[dynamodb.UpdateItemInput, dynamodb.UpdateItemOutput]
}

func NewMockClient(t *testing.T) *MockClient
```

#### LocalDynamoDB

```go
type LocalDynamoDB struct {
    Client *dynamodb.Client
    Port   int
}

func NewLocalClient(port int) *dynamodb.Client
func NewDefaultLocalClient() *dynamodb.Client
func WithDefaultLocalDynamoDB(t *testing.T, fn func(*LocalDynamoDB))
func WithIsolatedTable(t *testing.T, client *dynamodb.Client, fn func(string))
```

#### EntityBuilder

```go
type EntityBuilder struct {
    // Internal fields
}

type EntityOption func(*EntityBuilder)

func NewEntity() *EntityBuilder
func NewEntity(opts ...EntityOption) *EntityBuilder
func (b *EntityBuilder) Build() *TestEntity

// Functional options
func WithID(id string) EntityOption
func WithPrefix(prefix string) EntityOption
func WithLabel(label string) EntityOption
func WithData(data interface{}) EntityOption
func WithRelationship(name string, entity dynamap.Marshaler) EntityOption
func WithRelationships(name string, entities ...dynamap.Marshaler) EntityOption
// ... and more
```

#### SeedTestData

```go
type SeedTestData struct {
    // Internal fields
}

func NewSeedTestData(client *dynamodb.Client, tableName string) *SeedTestData
func (s *SeedTestData) SeedEntity(ctx context.Context, entity *TestEntity) error
func (s *SeedTestData) SeedFromJSON(ctx context.Context, r io.Reader) (int, error)
```

#### TestEntity

```go
type TestEntity struct {
    // Internal fields - implements all dynamap interfaces
}

// Implements:
// - dynamap.Marshaler
// - dynamap.RefMarshaler
// - dynamap.Unmarshaler
// - dynamap.RefUnmarshaler
```

### Helper Functions

```go
// Table management
func NewTestTable(prefix string) string
func NewTableManager(client *dynamodb.Client) *TableManager

// Utility functions
func IsLocalDynamoDBAvailable(port int) bool
func WaitForLocalDynamoDB(port int, timeout time.Duration) error
```

## Examples

### Complete Integration Test

```go
func TestCompleteOrderWorkflow(t *testing.T) {
    dynamock.WithDefaultLocalDynamoDB(t, func(local *dynamock.LocalDynamoDB) {
        dynamock.WithIsolatedTable(t, local.Client, func(tableName string) {
            seeder := dynamock.NewSeedTestData(local.Client, tableName)

            // Seed initial data from JSON
            file, err := os.Open("testdata/ecommerce.json")
            if err != nil {
                t.Skipf("Test data not found: %v", err)
            }
            defer file.Close()

            count, err := seeder.SeedFromJSON(context.Background(), file)
            if err != nil {
                t.Fatalf("Failed to seed data: %v", err)
            }

            t.Logf("Seeded %d entities", count)

            // Add additional test data using builders
            newProduct := dynamock.NewEntity(
                dynamock.WithID("P999"),
                dynamock.WithPrefix("product"),
                dynamock.WithLabel("product"),
                dynamock.WithData(map[string]interface{}{
                    "name":     "Special Product",
                    "category": "limited",
                    "price":    999,
                }),
            ).Build()

            err = seeder.SeedEntity(context.Background(), newProduct)
            if err != nil {
                t.Fatalf("Failed to seed additional product: %v", err)
            }

            // Test your application logic here
            table := dynamap.NewTable(tableName)

            // Example: Query for products
            queryList := &dynamap.QueryList{
                Label: "product",
                Limit: 10,
            }

            queryInput, err := table.MarshalQuery(queryList)
            if err != nil {
                t.Fatalf("Failed to marshal query: %v", err)
            }

            result, err := local.Client.Query(context.Background(), queryInput)
            if err != nil {
                t.Fatalf("Failed to query: %v", err)
            }

            // Use assertions to verify results
            assert.Items(t, result.Items).
                IsNotEmpty().
                ContainsEntityWithLabel("product")
        })
    })
}
```

### Mock Testing with Error Simulation

```go
func TestErrorHandling(t *testing.T) {
    mock := dynamock.NewMockClient(t)

    // Simulate conditional check failure
    mock.PutFunc = func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
        return nil, &types.ConditionalCheckFailedException{
            Message: aws.String("Item already exists"),
        }
    }

    table := dynamap.NewTable("test-table")
    entity := dynamock.NewEntity(
        dynamock.WithID("E1"),
        dynamock.WithPrefix("entity"),
    ).Build()

    putInput, err := table.MarshalPut(entity)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    _, err = mock.PutItem(context.Background(), putInput)

    // Verify error handling
    var condErr *types.ConditionalCheckFailedException
    if !errors.As(err, &condErr) {
        t.Errorf("Expected ConditionalCheckFailedException, got %T", err)
    }
}
```

## Best Practices

### Test Organization

1. **Use isolated tables** for each test to prevent interference
2. **Group related tests** in the same test file
3. **Use descriptive test names** that explain the scenario being tested
4. **Clean up resources** using defer statements or helper functions

### Mock vs Integration Testing

**Use Mock Testing When:**

- Testing business logic in isolation
- Simulating error conditions
- Fast unit tests
- Testing edge cases

**Use Integration Testing When:**

- Testing end-to-end workflows
- Validating DynamoDB queries and indexes
- Testing data consistency
- Performance testing

### Test Data Management

1. **Use JSON seeding** for complex, reusable test scenarios
2. **Use builders** for dynamic or programmatic test data
3. **Keep test data files** in version control
4. **Use descriptive entity IDs** for easier debugging

### Performance Considerations

1. **Reuse LocalDynamoDB instances** when possible
2. **Use batch operations** for seeding large datasets
3. **Limit test data size** to what's necessary for the test
4. **Run integration tests in parallel** with isolated tables

### Error Handling

1. **Test both success and failure paths**
2. **Use specific error types** in assertions
3. **Provide clear error messages** in test failures
4. **Handle setup failures gracefully** (skip tests when DynamoDB Local unavailable)

### CI/CD Integration

1. **Use environment detection** to skip integration tests when appropriate
2. **Set up DynamoDB Local** in CI pipelines
3. **Use unique table names** to support parallel test execution
4. **Clean up resources** to prevent resource leaks

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test -v ./dynamock/...`
5. Update documentation as needed
6. Submit a pull request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/nisimpson/dynamap.git
cd dynamap/dynamock

# Run tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires DynamoDB Local)
docker run -p 8000:8000 amazon/dynamodb-local
go test -v ./... -run Integration
```
