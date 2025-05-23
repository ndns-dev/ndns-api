package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sh5080/ndns-go/pkg/configs"
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// ServerStatusRepository는 서버 상태 정보를 DynamoDB에 저장하고 조회하는 레포지토리입니다.
type ServerStatusRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewServerStatusRepository는 새로운 서버 상태 레포지토리를 생성합니다.
func NewServerStatusRepository() (*ServerStatusRepository, error) {
	cfg := configs.GetConfig()
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWS.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS.AccessKeyID,
			cfg.AWS.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("AWS 설정 로드 실패: %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(awsCfg)

	repo := &ServerStatusRepository{
		client:    dynamoClient,
		tableName: model.TableNameServerStatus,
	}

	if err := repo.CreateServerStatusTableIfNotExists(); err != nil {
		return nil, fmt.Errorf("서버 상태 테이블 생성 실패: %v", err)
	}

	return repo, nil
}

// CreateServerStatusTableIfNotExists는 서버 상태 테이블이 없을 경우 생성합니다.
func (r *ServerStatusRepository) CreateServerStatusTableIfNotExists() error {
	// 테이블 존재 여부 확인
	exists, err := r.tableExists()
	if err != nil {
		return fmt.Errorf("테이블 존재 여부 확인 실패: %v", err)
	}

	// 테이블이 이미 존재하면 생성하지 않음
	if exists {
		return nil
	}

	// 테이블 생성 요청
	_, err = r.client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName: aws.String(r.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("AppName"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("AppName"),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return fmt.Errorf("테이블 생성 실패: %v", err)
	}

	// 테이블 생성 완료될 때까지 대기
	waiter := dynamodb.NewTableExistsWaiter(r.client)
	err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	}, 2*time.Minute)

	if err != nil {
		return fmt.Errorf("테이블 생성 완료 대기 실패: %v", err)
	}

	return nil
}

// tableExists는 테이블이 존재하는지 확인합니다.
func (r *ServerStatusRepository) tableExists() (bool, error) {
	_, err := r.client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	})

	if err != nil {
		// 테이블이 존재하지 않는 경우
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// UpdateServerStatus는 서버 상태를 업데이트합니다.
func (r *ServerStatusRepository) UpdateServerStatus(status *model.ServerStatus) error {
	// DynamoDB에 저장할 수 있도록 마샬링
	item, err := attributevalue.MarshalMap(status)
	if err != nil {
		return fmt.Errorf("서버 상태 마샬 실패: %v", err)
	}

	// DynamoDB에 저장
	_, err = r.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("서버 상태 저장 실패: %v", err)
	}

	return nil
}

// GetServerStatus는 특정 AppName에 대한 서버 상태를 조회합니다.
func (r *ServerStatusRepository) GetServerStatus(appName string) (*model.ServerStatus, error) {
	result, err := r.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"AppName": &types.AttributeValueMemberS{Value: appName},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("서버 상태 조회 실패: %v", err)
	}

	// 결과가 없는 경우
	if result.Item == nil {
		return nil, nil
	}

	var status model.ServerStatus
	err = attributevalue.UnmarshalMap(result.Item, &status)
	if err != nil {
		return nil, fmt.Errorf("서버 상태 언마샬 실패: %v", err)
	}

	return &status, nil
}

// GetAllServerStatuses는 모든 서버 상태 정보를 조회합니다.
func (r *ServerStatusRepository) GetAllServerStatuses() ([]*model.ServerStatus, error) {
	result, err := r.client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	})

	if err != nil {
		return nil, fmt.Errorf("서버 상태 조회 실패: %v", err)
	}

	var statuses []*model.ServerStatus
	err = attributevalue.UnmarshalListOfMaps(result.Items, &statuses)
	if err != nil {
		return nil, fmt.Errorf("서버 상태 언마샬 실패: %v", err)
	}

	// 만료된 항목 필터링
	now := time.Now()
	var validStatuses []*model.ServerStatus
	for _, status := range statuses {
		if status.ExpiresAt.After(now) {
			validStatuses = append(validStatuses, status)
		}
	}

	return validStatuses, nil
}

// DeleteServerStatus는 특정 AppName에 대한 서버 상태 정보를 삭제합니다.
func (r *ServerStatusRepository) DeleteServerStatus(appName string) error {
	_, err := r.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"AppName": &types.AttributeValueMemberS{Value: appName},
		},
	})

	if err != nil {
		return fmt.Errorf("서버 상태 삭제 실패: %v", err)
	}

	return nil
}
