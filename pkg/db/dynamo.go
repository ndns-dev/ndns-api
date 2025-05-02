package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sh5080/ndns-go/pkg/configs"
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// DynamoDBService는 DynamoDB와의 상호작용을 관리하는 서비스입니다.
type DynamoDBService struct {
	client    *dynamodb.Client
	tableName string
	config    *configs.EnvConfig
}

// NewDynamoDBService는 새로운 DynamoDB 서비스를 생성합니다.
func NewDynamoDBService(config *configs.EnvConfig) (*DynamoDBService, error) {
	// AWS 설정
	var cfg aws.Config
	var err error

	// AWS 자격증명이 설정되어 있을 경우
	if config.AWS.AccessKeyID != "" && config.AWS.SecretAccessKey != "" {
		// 고정 자격증명 사용
		creds := credentials.NewStaticCredentialsProvider(
			config.AWS.AccessKeyID,
			config.AWS.SecretAccessKey,
			"",
		)

		// AWS 설정 먼저 로드
		cfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(config.AWS.Region),
			awsconfig.WithCredentialsProvider(creds),
		)
	} else {
		// 기본 자격증명 프로바이더 체인 사용
		cfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(config.AWS.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("AWS 설정 로드 실패: %v", err)
	}

	// 그 다음 DynamoDB 클라이언트 생성 시 옵션 추가
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if config.AWS.DynamoDBEndpoint != "" {
			o.EndpointResolver = dynamodb.EndpointResolverFromURL(config.AWS.DynamoDBEndpoint)
		}
	})
	tableName := config.AWS.Tables.OCRCache

	return &DynamoDBService{
		client:    client,
		tableName: tableName,
		config:    config,
	}, nil
}

// CreateTableIfNotExists는 OCR 캐시 테이블이 없을 경우 생성합니다.
func (s *DynamoDBService) CreateTableIfNotExists() error {
	// 테이블 존재 여부 확인
	exists, err := s.tableExists()
	if err != nil {
		return fmt.Errorf("테이블 존재 여부 확인 실패: %v", err)
	}

	// 테이블이 이미 존재하면 생성하지 않음
	if exists {
		return nil
	}

	// 테이블 생성 요청
	_, err = s.client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName: aws.String(s.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("ImageURL"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("ImageURL"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return fmt.Errorf("테이블 생성 실패: %v", err)
	}

	// 테이블 생성 완료될 때까지 대기
	waiter := dynamodb.NewTableExistsWaiter(s.client)
	err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	}, 2*time.Minute)

	if err != nil {
		return fmt.Errorf("테이블 생성 완료 대기 실패: %v", err)
	}

	return nil
}

// tableExists는 테이블이 존재하는지 확인합니다.
func (s *DynamoDBService) tableExists() (bool, error) {
	_, err := s.client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		// 테이블이 존재하지 않는 경우
		var notFoundErr *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundErr); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetOCRCache는 이미지 URL로 OCR 캐시를 조회합니다.
func (s *DynamoDBService) GetOCRCache(imageURL string) (*model.OCRCache, error) {
	result, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"ImageURL": &types.AttributeValueMemberS{Value: imageURL},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("OCR 캐시 조회 실패: %v", err)
	}

	// 결과가 없는 경우
	if result.Item == nil {
		return nil, nil
	}

	var cache model.OCRCache
	err = attributevalue.UnmarshalMap(result.Item, &cache)
	if err != nil {
		return nil, fmt.Errorf("OCR 캐시 언마샬 실패: %v", err)
	}

	// 만료된 캐시인지 확인
	if time.Now().After(cache.ExpiresAt) {
		// 만료된 캐시 삭제 (비동기로 실행)
		go s.DeleteOCRCache(imageURL)
		return nil, nil
	}

	return &cache, nil
}

// SaveOCRCache는 OCR 결과를 캐시에 저장합니다.
func (s *DynamoDBService) SaveOCRCache(cache *model.OCRCache) error {
	// 현재 시간 설정
	now := time.Now()
	cache.CreatedAt = now

	// 만료 시간이 설정되지 않은 경우 기본값으로 7일 후로 설정
	if cache.ExpiresAt.IsZero() {
		cache.ExpiresAt = now.Add(7 * 24 * time.Hour)
	}

	// DynamoDB에 저장할 수 있도록 마샬링
	item, err := attributevalue.MarshalMap(cache)
	if err != nil {
		return fmt.Errorf("OCR 캐시 마샬 실패: %v", err)
	}

	// DynamoDB에 저장
	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("OCR 캐시 저장 실패: %v", err)
	}

	return nil
}

// DeleteOCRCache는 이미지 URL로 OCR 캐시를 삭제합니다.
func (s *DynamoDBService) DeleteOCRCache(imageURL string) error {
	_, err := s.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"ImageURL": &types.AttributeValueMemberS{Value: imageURL},
		},
	})

	if err != nil {
		return fmt.Errorf("OCR 캐시 삭제 실패: %v", err)
	}

	return nil
}
