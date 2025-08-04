# Dynamap

[![Test](https://github.com/nisimpson/dynamap/actions/workflows/test.yml/badge.svg)](https://github.com/nisimpson/dynamap/actions/workflows/test.yml)
[![GoDoc](https://godoc.org/github.com/nisimpson/dynamap?status.svg)](http://godoc.org/github.com/nisimpson/dynamap)
[![Release](https://img.shields.io/github/release/nisimpson/dynamap.svg)](https://github.com/nisimpson/dynamap/releases)

Dynamap is a lightweight entity-relationship abstraction layer over the AWS SDK for Go v2 DynamoDB client. It allows you to model relationships between domain objects and perform standard CRUD operations on a DynamoDB table with a consistent schema.

## Features

- **Entity-Relationship Modeling**: Define relationships between domain objects with type-safe operations
- **Single-Table Design**: Uses a consistent DynamoDB schema with delimited keys (e.g., `order#O1`)
- **Functional Options**: Clean API using functional options pattern for configuration
- **Query System**: Two main query types - `QueryList` for collections and `QueryEntity` for entity relationships
- **Built-in Pagination**: Cursor-based pagination with automatic storage in the same table
- **Comprehensive Error Handling**: Structured error handling without custom error types
- **Context Propagation**: Full context support for all operations
- **Testable Design**: Dependency injection for time functions and configurable backends
- **High Test Coverage**: High test coverage with comprehensive examples

## Table of Contents

- [Installation](#installation)
- [DynamoDB Table Setup](#dynamodb-table-setup)
  - [Table Schema](#table-schema)
  - [Required Global Secondary Index](#required-global-secondary-index)
  - [Creating the Table](#creating-the-table)
  - [IAM Permissions](#iam-permissions)
  - [Cost Considerations](#cost-considerations)
  - [Example Data Layout](#example-data-layout)
  - [Connecting to Your Table](#connecting-to-your-table)
- [Quick Start](#quick-start)
  - [Define Your Entities](#define-your-entities)
  - [Basic Operations](#basic-operations)
- [Core Concepts](#core-concepts)
  - [Table Configuration](#table-configuration)
  - [DynamoDB Schema](#dynamodb-schema)
  - [Interfaces](#interfaces)
- [Advanced Usage](#advanced-usage)
  - [Query System](#query-system)
  - [Pagination](#pagination)
  - [Functional Options](#functional-options)
  - [Custom Key Delimiters](#custom-key-delimiters)
  - [Time-to-Live Support](#time-to-live-support)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Installation

```bash
go get github.com/nisimpson/dynamap
```

## DynamoDB Table Setup

Dynamap uses a single-table design that requires a specific DynamoDB schema to function properly.

### Table Schema

The table uses the following attributes:

| Attribute    | Type   | Purpose                                                     |
| ------------ | ------ | ----------------------------------------------------------- |
| `hk`         | String | **Partition Key** - Source entity key (format: `prefix#id`) |
| `sk`         | String | **Sort Key** - Target entity key (format: `prefix#id`)      |
| `label`      | String | **GSI Partition Key** - Relationship label/category         |
| `gsi1_sk`    | String | **GSI Sort Key** - Custom sort key for the relationship     |
| `data`       | Map    | Entity data (JSON)                                          |
| `created_at` | String | ISO 8601 timestamp                                          |
| `updated_at` | String | ISO 8601 timestamp                                          |
| `expires`    | Number | TTL expiration (Unix timestamp)                             |

### Required Global Secondary Index

The table requires one Global Secondary Index (GSI):

- **Index Name**: `ref-index` (configurable via `Table.RefIndexName`)
- **Partition Key**: `label`
- **Sort Key**: `gsi1_sk`
- **Projection**: `ALL` (recommended) or `INCLUDE` with required attributes

### Creating the Table

#### Option 1: AWS CLI

```bash
# Create the main table
aws dynamodb create-table \
    --table-name my-app-table \
    --attribute-definitions \
        AttributeName=hk,AttributeType=S \
        AttributeName=sk,AttributeType=S \
        AttributeName=label,AttributeType=S \
        AttributeName=gsi1_sk,AttributeType=S \
    --key-schema \
        AttributeName=hk,KeyType=HASH \
        AttributeName=sk,KeyType=RANGE \
    --global-secondary-indexes \
        'IndexName=ref-index,KeySchema=[{AttributeName=label,KeyType=HASH},{AttributeName=gsi1_sk,KeyType=RANGE}],Projection={ProjectionType=ALL},ProvisionedThroughput={ReadCapacityUnits=5,WriteCapacityUnits=5}' \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --region us-east-1

# Enable TTL on the expires attribute (optional)
aws dynamodb update-time-to-live \
    --table-name my-app-table \
    --time-to-live-specification \
        Enabled=true,AttributeName=expires \
    --region us-east-1
```

#### Option 2: AWS CDK (TypeScript)

```typescript
import {
  Table,
  AttributeType,
  ProjectionType,
  BillingMode,
} from "aws-cdk-lib/aws-dynamodb";

const table = new Table(this, "MyAppTable", {
  tableName: "my-app-table",
  partitionKey: { name: "hk", type: AttributeType.STRING },
  sortKey: { name: "sk", type: AttributeType.STRING },
  timeToLiveAttribute: "expires",
  billingMode: BillingMode.PAY_PER_REQUEST, // or PROVISIONED
});

table.addGlobalSecondaryIndex({
  indexName: "ref-index",
  partitionKey: { name: "label", type: AttributeType.STRING },
  sortKey: { name: "gsi1_sk", type: AttributeType.STRING },
  projectionType: ProjectionType.ALL,
});
```

#### Option 3: Terraform

```hcl
resource "aws_dynamodb_table" "my_app_table" {
  name           = "my-app-table"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "hk"
  range_key      = "sk"

  attribute {
    name = "hk"
    type = "S"
  }

  attribute {
    name = "sk"
    type = "S"
  }

  attribute {
    name = "label"
    type = "S"
  }

  attribute {
    name = "gsi1_sk"
    type = "S"
  }

  global_secondary_index {
    name            = "ref-index"
    hash_key        = "label"
    range_key       = "gsi1_sk"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "expires"
    enabled        = true
  }

  tags = {
    Name = "MyAppTable"
  }
}
```

#### Option 4: CloudFormation

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Description: "DynamoDB table for Dynamap library"

Resources:
  MyAppTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: my-app-table
      BillingMode: PAY_PER_REQUEST
      AttributeDefinitions:
        - AttributeName: hk
          AttributeType: S
        - AttributeName: sk
          AttributeType: S
        - AttributeName: label
          AttributeType: S
        - AttributeName: gsi1_sk
          AttributeType: S
      KeySchema:
        - AttributeName: hk
          KeyType: HASH
        - AttributeName: sk
          KeyType: RANGE
      GlobalSecondaryIndexes:
        - IndexName: ref-index
          KeySchema:
            - AttributeName: label
              KeyType: HASH
            - AttributeName: gsi1_sk
              KeyType: RANGE
          Projection:
            ProjectionType: ALL
      TimeToLiveSpecification:
        AttributeName: expires
        Enabled: true
      Tags:
        - Key: Name
          Value: MyAppTable

Outputs:
  TableName:
    Description: "Name of the DynamoDB table"
    Value: !Ref MyAppTable
    Export:
      Name: !Sub "${AWS::StackName}-TableName"
```

### IAM Permissions

Your application needs the following DynamoDB permissions:

#### Minimal IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:BatchGetItem",
        "dynamodb:BatchWriteItem"
      ],
      "Resource": [
        "arn:aws:dynamodb:REGION:ACCOUNT:table/my-app-table",
        "arn:aws:dynamodb:REGION:ACCOUNT:table/my-app-table/index/ref-index"
      ]
    }
  ]
}
```

#### Complete IAM Policy (with pagination support)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:BatchGetItem",
        "dynamodb:BatchWriteItem"
      ],
      "Resource": [
        "arn:aws:dynamodb:REGION:ACCOUNT:table/my-app-table",
        "arn:aws:dynamodb:REGION:ACCOUNT:table/my-app-table/index/ref-index"
      ]
    }
  ]
}
```

**Note**: Replace `REGION` with your AWS region (e.g., `us-east-1`) and `ACCOUNT` with your AWS account ID.

### Cost Considerations

- **On-Demand Billing**: Recommended for most applications with unpredictable traffic
- **Provisioned Billing**: Better for predictable, consistent workloads
- **GSI Costs**: The GSI will incur additional read/write costs
- **Storage**: Single-table design is generally storage-efficient
- **TTL**: Use TTL to automatically clean up expired relationships and reduce storage costs

### Example Data Layout

Here's how your data will look in DynamoDB:

| hk           | sk           | label               | gsi1_sk       | data             | created_at             |
| ------------ | ------------ | ------------------- | ------------- | ---------------- | ---------------------- |
| `order#O1`   | `order#O1`   | `order`             | `2025-01-01`  | `{"id":"O1"...}` | `2025-01-01T10:00:00Z` |
| `order#O1`   | `product#P1` | `order/O1/products` | `electronics` | `{"id":"P1"...}` | `2025-01-01T10:00:00Z` |
| `order#O1`   | `product#P2` | `order/O1/products` | `books`       | `{"id":"P2"...}` | `2025-01-01T10:00:00Z` |
| `product#P1` | `product#P1` | `product`           | `electronics` | `{"id":"P1"...}` | `2025-01-01T09:30:00Z` |

This design allows for:

- **Entity queries**: Find an order and all its products
- **Collection queries**: Find all products in a category
- **Relationship traversal**: Navigate between related entities
- **Efficient pagination**: Using built-in DynamoDB pagination

### Connecting to Your Table

Once your table is created, connect Dynamap to it:

```go
// Use the same table name you created above
table := dynamap.NewTable("my-app-table")

// Optional: customize the index name if you used a different name
table.RefIndexName = "custom-ref-index"

// Optional: customize delimiters
table.KeyDelimiter = "#"      // Default
table.LabelDelimiter = "/"    // Default
```

## Quick Start

### Define Your Entities

```go
// Order implements dynamap.RefMarshaler
type Order struct {
    ID          string    `dynamodbav:"id"`
    PurchasedBy string    `dynamodbav:"purchased_by"`
    Products    []Product `dynamodbav:"-"`
    Created     time.Time `dynamodbav:"-"`
}

func (o *Order) MarshalSelf(opts *dynamap.MarshalOptions) error {
    opts.WithSelfTarget("order", o.ID)
    opts.Created = o.Created
    opts.RefSortKey = opts.Created.Format(time.RFC3339)
    return nil
}

func (o *Order) MarshalRefs(ctx *dynamap.RelationshipContext) error {
    // Convert Product slice to Marshaler slice
    productPtrs := make([]*Product, len(o.Products))
    for i := range o.Products {
        productPtrs[i] = &o.Products[i]
    }
    ctx.AddMany("products", dynamap.SliceOf(productPtrs...))
    return nil
}

func (o *Order) UnmarshalSelf(rel *dynamap.Relationship) error {
    o.Created = rel.CreatedAt
    return nil
}

func (o *Order) UnmarshalRef(name string, id string, ref *dynamap.Relationship) error {
    if name == "products" {
        var product Product
        product.ID = id
        if err := product.UnmarshalSelf(ref); err != nil {
            return err
        }
        o.Products = append(o.Products, product)
    }
    return nil
}

// Product implements dynamap.Marshaler
type Product struct {
    ID       string `dynamodbav:"id"`
    Category string `dynamodbav:"category"`
}

func (p *Product) MarshalSelf(opts *dynamap.MarshalOptions) error {
    opts.WithSelfTarget("product", p.ID)
    opts.RefSortKey = p.Category
    return nil
}
```

### Basic Operations

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/nisimpson/dynamap"
)

func main() {
    ctx := context.Background()
    cfg, _ := config.LoadDefaultConfig(ctx)
    ddb := dynamodb.NewFromConfig(cfg)

    // Create table configuration
    table := dynamap.NewTable("my-table")

    // Store a single entity
    product := &Product{ID: "P1", Category: "electronics"}
    putInput, err := table.MarshalPut(product)
    if err != nil {
        log.Fatal(err)
    }

    _, err = ddb.PutItem(ctx, putInput)
    if err != nil {
        log.Fatal(err)
    }

    // Store an entity with relationships using batch operations
    order := &Order{
        ID:          "O1",
        PurchasedBy: "joe",
        Products:    []Product{*product},
    }

    batches, err := table.MarshalBatch(order)
    if err != nil {
        log.Fatal(err)
    }

    for _, batch := range batches {
        _, err = ddb.BatchWriteItem(ctx, batch)
        if err != nil {
            log.Fatal(err)
        }
    }

    // Query for an entity and its relationships
    queryEntity := &dynamap.QueryEntity{
        Source: &Order{ID: "O1"},
        Limit:  50,
    }

    queryInput, err := table.MarshalQuery(queryEntity)
    if err != nil {
        log.Fatal(err)
    }

    result, err := ddb.Query(ctx, queryInput)
    if err != nil {
        log.Fatal(err)
    }

    // Unmarshal results
    var fullOrder Order
    relationships, err := dynamap.UnmarshalEntity(result.Items, &fullOrder)
    if err != nil {
        if errors.Is(err, dynamap.ErrItemNotFound) {
            log.Println("Order not found")
        } else {
            log.Fatal(err)
        }
    }

    fmt.Printf("Order has %d relationships\n", len(relationships))
}
```

## Core Concepts

### Table Configuration

The `Table` struct provides simple configuration:

```go
// Basic table configuration
table := dynamap.NewTable("my-table")

// Custom configuration
table.KeyDelimiter = "|"            // Default: "#"
table.RefIndexName = "custom-index" // Default: "ref-index"
table.PaginationTTL = time.Hour     // Default: 24 hours
```

### DynamoDB Schema

The library uses a single-table design with delimited keys:

- **Main Table**: `hk` (partition key), `sk` (sort key)
- **Ref Index**: `label` (partition key), `gsi1_sk` (sort key)

Example data:

```
| hk       | sk         | label             | gsi1_sk     |
|----------|------------|-------------------|-------------|
| order#O1 | order#O1   | order             | 2025-01-01  |
| order#O1 | product#P1 | order/O1/products | electronics |
| order#O1 | product#P2 | order/O1/products | books       |
```

### Interfaces

- **`Marshaler`**: Entities that can be marshaled into relationships
- **`RefMarshaler`**: Entities that can marshal both themselves and their relationships
- **`Unmarshaler`**: Entities that can extract data from relationships
- **`RefUnmarshaler`**: Entities that can extract data from any relationship

## Advanced Usage

### Query System

```go
// Query all products by label
queryList := &dynamap.QueryList{
    Label: "product",
    LabelSortFilter: func() *expression.KeyConditionBuilder {
        condition := expression.Key(AttributeNameRefSortKey).BeginsWith("electronics")
        return &condition
    }(),
    Limit: 10,
}

queryInput, err := table.MarshalQuery(queryList)
result, err := ddb.Query(ctx, queryInput)

// Query an entity and its relationships
queryEntity := &dynamap.QueryEntity{
    Source: &Order{ID: "O1"},
    TargetFilter: expression.Key(AttributeNameTarget).BeginsWith("product#"),
    Limit: 20,
}
```

### Pagination

```go
// Create paginator
paginator := table.Paginator(ddb)

// Generate cursor from last evaluated key
cursor, err := dynamap.MarshalStartKey(ctx, paginator, result.LastEvaluatedKey)

// Use cursor in next query
startKey, err := dynamap.UnmarshalStartKey(ctx, paginator, cursor)
queryList.StartKey = startKey
```

### Functional Options

```go
// Marshal with custom options
relationships, err := dynamap.MarshalRelationships(entity, func(opts *dynamap.MarshalOptions) {
    opts.TimeToLive = 30 * 24 * time.Hour // 30 days
    opts.Delimiter = "|"                   // Custom delimiter
})

// Table operations with options
putInput, err := table.MarshalPut(entity, func(opts *dynamap.MarshalOptions) {
    opts.Created = time.Now()
    opts.Updated = time.Now()
})
```

### Custom Key Delimiters

```go
table := dynamap.NewTable("my-table")
table.KeyDelimiter = "|"

// Keys will be formatted as: product|P1, order|O1, etc.
```

### Time-to-Live Support

```go
// Set TTL on relationships
relationships, err := dynamap.MarshalRelationships(entity, func(opts *dynamap.MarshalOptions) {
    opts.TimeToLive = 7 * 24 * time.Hour // 7 days
})
```

## Error Handling

The library uses standard Go error handling without custom error types:

```go
_, err := dynamap.UnmarshalEntity(items, &entity)
if err != nil {
    if errors.Is(err, dynamap.ErrItemNotFound) {
        // Handle item not found
    } else {
        // Handle other errors
        log.Printf("Error: %v", err)
    }
}
```

## Testing

The library includes comprehensive tests with over 90% coverage:

```bash
go test -v                    # Run all tests
go test -cover               # Run with coverage
go test -run TestExample     # Run specific tests
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for your changes
4. Ensure all tests pass: `go test -v`
5. Ensure coverage stays above 90%: `go test -cover`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
