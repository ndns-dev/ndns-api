package client

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/sh5080/ndns-go/pkg/configs"
)

// NewDynamoDBClient는 설정에 따라 DynamoDB 클라이언트를 생성합니다
func NewDynamoDBClient() (*dynamodb.Client, error) {
	config := configs.GetConfig()

	// AWS 설정
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config.AWS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AWS.AccessKeyId,
			config.AWS.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	// DynamoDB 클라이언트 생성
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if config.AWS.DynamoDBEndpoint != "" {
			o.EndpointResolver = dynamodb.EndpointResolverFromURL(config.AWS.DynamoDBEndpoint)
		}
	})

	return client, nil
}
