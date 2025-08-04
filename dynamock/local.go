package dynamock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// LocalDynamoDB represents a connection to a local DynamoDB instance.
type LocalDynamoDB struct {
	Client   *dynamodb.Client
	Endpoint string
	Port     int
}

// NewLocalClient creates a DynamoDB client configured to connect to a local DynamoDB instance.
// This is useful for integration testing with DynamoDB Local.
//
// Example usage:
//
//	client := dynamock.NewLocalClient(8000)
//	// Use client with your tests
func NewLocalClient(port int) *dynamodb.Client {
	endpoint := fmt.Sprintf("http://localhost:%d", port)

	cfg := aws.Config{
		Region:      "us-east-1", // DynamoDB Local doesn't care about region
		Credentials: aws.AnonymousCredentials{},
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: region,
				}, nil
			},
		),
	}

	return dynamodb.NewFromConfig(cfg)
}

// NewLocalDynamoDB creates a LocalDynamoDB instance with the specified port.
// This provides additional utilities beyond just the client.
func NewLocalDynamoDB(port int) *LocalDynamoDB {
	endpoint := fmt.Sprintf("http://localhost:%d", port)
	client := NewLocalClient(port)

	return &LocalDynamoDB{
		Client:   client,
		Endpoint: endpoint,
		Port:     port,
	}
}

// IsAvailable checks if DynamoDB Local is running on the configured port.
func (l *LocalDynamoDB) IsAvailable(ctx context.Context) bool {
	// Try to connect to the port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", l.Port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	// Try to list tables to verify it's actually DynamoDB
	_, err = l.Client.ListTables(ctx, &dynamodb.ListTablesInput{})
	return err == nil
}

// WaitForAvailable waits for DynamoDB Local to become available.
// Returns an error if it doesn't become available within the timeout.
func (l *LocalDynamoDB) WaitForAvailable(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if l.IsAvailable(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// Continue checking
		}
	}

	return fmt.Errorf("DynamoDB Local not available at %s after %v", l.Endpoint, timeout)
}

// CreateDynamapTable creates a table with the standard dynamap schema.
// This is a convenience function for integration tests.
func (l *LocalDynamoDB) CreateDynamapTable(ctx context.Context, tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("hk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("label"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("gsi1_sk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("hk"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       types.KeyTypeRange,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("ref-index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("label"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("gsi1_sk"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := l.Client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	// Wait for table to become active
	return l.WaitForTableActive(ctx, tableName, 30*time.Second)
}

// WaitForTableActive waits for a table to become active.
func (l *LocalDynamoDB) WaitForTableActive(ctx context.Context, tableName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		output, err := l.Client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return fmt.Errorf("failed to describe table %s: %w", tableName, err)
		}

		if output.Table.TableStatus == types.TableStatusActive {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue checking
		}
	}

	return fmt.Errorf("table %s did not become active within %v", tableName, timeout)
}

// DeleteTable deletes a table and waits for it to be fully deleted.
func (l *LocalDynamoDB) DeleteTable(ctx context.Context, tableName string) error {
	_, err := l.Client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete table %s: %w", tableName, err)
	}

	// Wait for table to be deleted
	return l.WaitForTableDeleted(ctx, tableName, 30*time.Second)
}

// WaitForTableDeleted waits for a table to be fully deleted.
func (l *LocalDynamoDB) WaitForTableDeleted(ctx context.Context, tableName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := l.Client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})

		// If we get a ResourceNotFoundException, the table is deleted
		if err != nil {
			var notFoundErr *types.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				return nil
			}
			return fmt.Errorf("error checking table deletion status: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue checking
		}
	}

	return fmt.Errorf("table %s was not deleted within %v", tableName, timeout)
}

// ListTables returns all table names in the local DynamoDB instance.
func (l *LocalDynamoDB) ListTables(ctx context.Context) ([]string, error) {
	output, err := l.Client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	return output.TableNames, nil
}

// Cleanup deletes all tables in the local DynamoDB instance.
// This is useful for cleaning up after integration tests.
func (l *LocalDynamoDB) Cleanup(ctx context.Context) error {
	tables, err := l.ListTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tables for cleanup: %w", err)
	}

	for _, tableName := range tables {
		if err := l.DeleteTable(ctx, tableName); err != nil {
			return fmt.Errorf("failed to delete table %s during cleanup: %w", tableName, err)
		}
	}

	return nil
}

// NewLocalClientFromConfig creates a local DynamoDB client using the provided AWS config.
// This allows for more customization than NewLocalClient.
func NewLocalClientFromConfig(cfg aws.Config, port int) *dynamodb.Client {
	endpoint := fmt.Sprintf("http://localhost:%d", port)

	// Override the endpoint resolver
	cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: region,
			}, nil
		},
	)

	// Use anonymous credentials for local testing
	cfg.Credentials = aws.AnonymousCredentials{}

	return dynamodb.NewFromConfig(cfg)
}

// MustNewLocalClient creates a local DynamoDB client and panics if it fails.
// This is useful for test setup where you want to fail fast.
func MustNewLocalClient(port int) *dynamodb.Client {
	client := NewLocalClient(port)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect to DynamoDB Local on port %d: %v", port, err))
	}

	return client
}

// DefaultLocalPort is the default port for DynamoDB Local.
const DefaultLocalPort = 8000

// NewDefaultLocalClient creates a local DynamoDB client using the default port (8000).
func NewDefaultLocalClient() *dynamodb.Client {
	return NewLocalClient(DefaultLocalPort)
}

// NewDefaultLocalDynamoDB creates a LocalDynamoDB instance using the default port (8000).
func NewDefaultLocalDynamoDB() *LocalDynamoDB {
	return NewLocalDynamoDB(DefaultLocalPort)
}
