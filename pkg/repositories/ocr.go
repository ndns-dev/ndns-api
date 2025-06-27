package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// OcrRepositoryImpl는 Ocr 작업 상태를 관리하는 리포지토리입니다
type OcrRepositoryImpl struct {
	// DynamoDB 클라이언트
	client      *dynamodb.Client
	tableName   string
	config      *configs.EnvConfig
	ocrState    map[string]*model.OcrQueueState
	jobsLock    sync.RWMutex
	ocrResult   map[string]*model.OcrResult
	resultsLock sync.RWMutex
}

// NewOcrRepository는 새 Ocr 저장소를 생성합니다
func NewOcrRepository() _interface.OcrRepository {
	config := configs.GetConfig()
	// AWS 설정
	cfg, _ := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(config.AWS.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AWS.AccessKeyId,
			config.AWS.SecretAccessKey,
			"",
		)),
	)

	// DynamoDB 클라이언트 생성
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if config.AWS.DynamoDBEndpoint != "" {
			o.EndpointResolver = dynamodb.EndpointResolverFromURL(config.AWS.DynamoDBEndpoint)
		}
	})

	repo := &OcrRepositoryImpl{
		client:    client,
		tableName: config.AWS.Tables.OcrResult,
		config:    config,
		ocrState:  make(map[string]*model.OcrQueueState),
		ocrResult: make(map[string]*model.OcrResult),
	}

	// 테이블 생성 확인
	if err := repo.createTableIfNotExists(); err != nil {
		return nil
	}

	return repo
}

// createTableIfNotExists는 필요한 DynamoDB 테이블을 생성합니다
func (r *OcrRepositoryImpl) createTableIfNotExists() error {
	_, err := r.client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	})
	if err == nil {
		return nil // 테이블이 이미 존재
	}

	// 테이블 생성
	_, err = r.client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		TableName: aws.String(r.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("ImageUrl"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("JobId"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("ImageUrl"),
				KeyType:       types.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("JobIdIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("JobId"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return fmt.Errorf("테이블 생성 실패: %v", err)
	}

	// 테이블 생성 완료 대기
	waiter := dynamodb.NewTableExistsWaiter(r.client)
	return waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	}, 2*time.Minute)
}

// SaveOcrJob은 새로운 Ocr 작업을 저장합니다
func (r *OcrRepositoryImpl) SaveOcrJob(queueState *model.OcrQueueState) error {
	// 메모리 캐시 업데이트
	r.jobsLock.Lock()
	r.ocrState[queueState.JobId] = queueState
	r.jobsLock.Unlock()

	// DynamoDB 업데이트
	crawlResultJSON, _ := json.Marshal(queueState.CrawlResult)
	_, err := r.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"JobId":           &types.AttributeValueMemberS{Value: queueState.JobId},
			"CrawlResult":     &types.AttributeValueMemberS{Value: string(crawlResultJSON)},
			"CurrentPosition": &types.AttributeValueMemberS{Value: string(queueState.CurrentPosition)},
			"Is2025OrLater":   &types.AttributeValueMemberBOOL{Value: queueState.Is2025OrLater},
			"RequestedAt":     &types.AttributeValueMemberS{Value: queueState.RequestedAt.Format(time.RFC3339)},
		},
	})

	return err
}

// GetOcrResult는 이미지 Url에 대한 Ocr 결과를 조회합니다
func (r *OcrRepositoryImpl) GetOcrResult(imageUrl string) (*model.OcrResult, error) {
	// 먼저 메모리 캐시 확인
	r.resultsLock.RLock()
	if result, exists := r.ocrResult[imageUrl]; exists {
		r.resultsLock.RUnlock()
		return result, nil
	}
	r.resultsLock.RUnlock()

	// DynamoDB에서 조회
	output, err := r.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ImageUrl": &types.AttributeValueMemberS{Value: imageUrl},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("DynamoDB 조회 실패: %v", err)
	}

	if output.Item == nil {
		return nil, nil
	}

	// 결과 파싱
	result := &model.OcrResult{
		ImageUrl: imageUrl,
		OcrText:  output.Item["OcrText"].(*types.AttributeValueMemberS).Value,
	}

	if v, ok := output.Item["ProcessedAt"].(*types.AttributeValueMemberS); ok {
		result.ProcessedAt, _ = time.Parse(time.RFC3339, v.Value)
	}

	// 메모리 캐시 업데이트
	r.resultsLock.Lock()
	r.ocrResult[imageUrl] = result
	r.resultsLock.Unlock()

	return result, nil
}

// SaveOcrResult는 Ocr 처리 결과를 저장합니다
func (r *OcrRepositoryImpl) SaveOcrResult(result *model.OcrResult) error {
	r.resultsLock.Lock()
	r.ocrResult[result.ImageUrl] = result
	r.resultsLock.Unlock()

	_, err := r.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"ImageUrl":    &types.AttributeValueMemberS{Value: result.ImageUrl},
			"JobId":       &types.AttributeValueMemberS{Value: result.JobId},
			"Position":    &types.AttributeValueMemberS{Value: string(result.Position)},
			"OcrText":     &types.AttributeValueMemberS{Value: result.OcrText},
			"ProcessedAt": &types.AttributeValueMemberS{Value: result.ProcessedAt.Format(time.RFC3339)},
			"Error":       &types.AttributeValueMemberS{Value: result.Error},
		},
	})

	return err
}

// GetOcrJob은 JobId에 해당하는 Ocr 작업 상태를 조회합니다
func (r *OcrRepositoryImpl) GetOcrJob(jobId string) (*model.OcrQueueState, error) {
	// 먼저 메모리 캐시 확인
	r.jobsLock.RLock()
	if job, exists := r.ocrState[jobId]; exists {
		r.jobsLock.RUnlock()
		return job, nil
	}
	r.jobsLock.RUnlock()

	// DynamoDB에서 조회
	output, err := r.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"JobId": &types.AttributeValueMemberS{Value: jobId},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("DynamoDB 조회 실패: %v", err)
	}

	if output.Item == nil {
		return nil, fmt.Errorf("ocr 작업을 찾을 수 없음: %s", jobId)
	}

	// 결과 파싱
	job := &model.OcrQueueState{
		JobId: jobId,
	}

	// 시간 정보 파싱
	if v, ok := output.Item["RequestedAt"].(*types.AttributeValueMemberS); ok {
		job.RequestedAt, _ = time.Parse(time.RFC3339, v.Value)
	}

	// 메모리 캐시 업데이트
	r.jobsLock.Lock()
	r.ocrState[jobId] = job
	r.jobsLock.Unlock()

	return job, nil
}
