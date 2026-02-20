package database

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// ConnectDynamoDB creates a DynamoDB client using environment variables.
//
// Supported env vars (local-friendly):
//   - AWS_REGION (default: us-east-1)
//   - AWS_ACCESS_KEY_ID (default: local)
//   - AWS_SECRET_ACCESS_KEY (default: local)
//   - DYNAMODB_ENDPOINT (optional; e.g. http://dynamodb:8000)
func ConnectDynamoDB() *dynamodb.Client {
	cfg, err := NewDynamoDBConfigFromEnv(context.Background())
	if err != nil {
		log.Fatalf("failed to create dynamodb config: %v", err)
	}
	return dynamodb.NewFromConfig(cfg)
}

func NewDynamoDBConfigFromEnv(ctx context.Context) (aws.Config, error) {
	region := getenvDefault("AWS_REGION", "us-east-1")
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")

	// Local DynamoDB does not validate credentials, but the AWS SDK requires them.
	creds := credentials.NewStaticCredentialsProvider(
		getenvDefault("AWS_ACCESS_KEY_ID", "local"),
		getenvDefault("AWS_SECRET_ACCESS_KEY", "local"),
		"",
	)

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	}

	if endpoint != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: endpoint, SigningRegion: region, HostnameImmutable: true}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		loadOpts = append(loadOpts, config.WithEndpointResolverWithOptions(resolver))
	}

	return config.LoadDefaultConfig(ctx, loadOpts...)
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
