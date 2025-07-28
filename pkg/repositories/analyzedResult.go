package repository

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	client "github.com/sh5080/ndns-go/pkg/clients"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// OcrRepositoryImpl는 Ocr 작업 상태를 관리하는 리포지토리입니다
type AnalyzedResultRepositoryImpl struct {
	// DynamoDB 클라이언트
	client    *dynamodb.Client
	tableName string
	config    *configs.EnvConfig
}

// NewAnalyzedResultRepository는 새 AnalyzedResult 저장소를 생성합니다
func NewAnalyzedResultRepository() _interface.AnalyzedResultRepository {
	config := configs.GetConfig()

	// 공통 DynamoDB 클라이언트 생성
	client, err := client.NewDynamoDBClient()
	if err != nil {
		return nil
	}

	repo := &AnalyzedResultRepositoryImpl{
		client:    client,
		tableName: config.AWS.Tables.AnalyzedResult,
		config:    config,
	}

	return repo
}

// GetAnalyzedResult는 링크에 대한 분석결과를 조회합니다
func (r *AnalyzedResultRepositoryImpl) GetAnalyzedResult(link string) (*model.AnalyzedResult, error) {
	// DynamoDB에서 분석결과 조회
	result, err := r.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"link": &types.AttributeValueMemberS{Value: link},
		},
	})
	if err != nil {
		return nil, err
	}

	// 조회된 결과가 없으면 nil 반환
	if result.Item == nil {
		return nil, nil
	}

	// 기본 필드들 파싱
	analyzedResult := &model.AnalyzedResult{
		Link: link,
	}

	// IsSponsored 파싱
	if isSponsoredVal, ok := result.Item["isSponsored"].(*types.AttributeValueMemberBOOL); ok {
		analyzedResult.IsSponsored = isSponsoredVal.Value
	}

	// SponsorProbability 파싱
	if probVal, ok := result.Item["sponsorProbability"].(*types.AttributeValueMemberN); ok {
		if prob, err := strconv.ParseFloat(probVal.Value, 64); err == nil {
			analyzedResult.SponsorProbability = prob
		}
	}

	// SponsorIndicators 파싱 (JSON 문자열에서)
	if indicatorsVal, ok := result.Item["sponsorIndicators"].(*types.AttributeValueMemberS); ok {
		var indicators []structure.SponsorIndicator
		if err := json.Unmarshal([]byte(indicatorsVal.Value), &indicators); err == nil {
			analyzedResult.SponsorIndicators = indicators
		}
	}

	return analyzedResult, nil
}

// GetAnalyzedResults는 여러 링크의 분석결과를 한 번에 조회합니다
func (r *AnalyzedResultRepositoryImpl) GetAnalyzedResults(links []string) (map[string]*model.AnalyzedResult, error) {
	if len(links) == 0 {
		return make(map[string]*model.AnalyzedResult), nil
	}

	utils.DebugLog("GetAnalyzedResults 시작: %d개 링크 조회\n", len(links))

	// DynamoDB BatchGetItem 요청을 위한 키 생성
	keys := make([]map[string]types.AttributeValue, len(links))
	for i, link := range links {
		keys[i] = map[string]types.AttributeValue{
			"link": &types.AttributeValueMemberS{Value: link},
		}
	}

	utils.DebugLog("DynamoDB BatchGetItem 요청 시작\n")
	// BatchGetItem 요청
	result, err := r.client.BatchGetItem(context.TODO(), &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			r.tableName: {Keys: keys},
		},
	})
	if err != nil {
		utils.DebugLog("BatchGetItem 실패: %v\n", err)
		return nil, err
	}

	utils.DebugLog("BatchGetItem 응답 받음\n")
	// 결과 매핑
	analyzedResults := make(map[string]*model.AnalyzedResult)
	if responses, exists := result.Responses[r.tableName]; exists {
		utils.DebugLog("테이블 %s에서 %d개 아이템 조회됨\n", r.tableName, len(responses))
		for _, item := range responses {
			// 기본 필드들 파싱
			analyzedResult := &model.AnalyzedResult{}

			// Link 파싱
			if linkVal, ok := item["link"].(*types.AttributeValueMemberS); ok {
				analyzedResult.Link = linkVal.Value
			}

			// IsSponsored 파싱
			if isSponsoredVal, ok := item["isSponsored"].(*types.AttributeValueMemberBOOL); ok {
				analyzedResult.IsSponsored = isSponsoredVal.Value
			}

			// SponsorProbability 파싱
			if probVal, ok := item["sponsorProbability"].(*types.AttributeValueMemberN); ok {
				if prob, err := strconv.ParseFloat(probVal.Value, 64); err == nil {
					analyzedResult.SponsorProbability = prob
				}
			}

			// SponsorIndicators 파싱 (JSON 문자열에서)
			if indicatorsVal, ok := item["sponsorIndicators"].(*types.AttributeValueMemberS); ok {
				var indicators []structure.SponsorIndicator
				if err := json.Unmarshal([]byte(indicatorsVal.Value), &indicators); err == nil {
					analyzedResult.SponsorIndicators = indicators
				} else {
					utils.DebugLog("SponsorIndicators JSON 파싱 실패: %v\n", err)
				}
			}

			analyzedResults[analyzedResult.Link] = analyzedResult
			utils.DebugLog("파싱된 결과: %s -> IsSponsored: %v\n", analyzedResult.Link, analyzedResult.IsSponsored)
		}
	} else {
		utils.DebugLog("테이블 %s에서 응답 없음\n", r.tableName)
	}

	utils.DebugLog("GetAnalyzedResults 완료: %d개 결과 반환\n", len(analyzedResults))
	return analyzedResults, nil
}

// SaveAnalyzedResult는 분석결과를 DynamoDB에 저장합니다
func (r *AnalyzedResultRepositoryImpl) SaveAnalyzedResult(result *model.AnalyzedResult) error {
	// SponsorIndicators를 JSON으로 직렬화
	indicatorsJSON, _ := json.Marshal(result.SponsorIndicators)

	// 분석결과 저장
	r.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"link":               &types.AttributeValueMemberS{Value: result.Link},
			"isSponsored":        &types.AttributeValueMemberBOOL{Value: result.IsSponsored},
			"sponsorProbability": &types.AttributeValueMemberN{Value: strconv.FormatFloat(result.SponsorProbability, 'f', -1, 64)},
			"sponsorIndicators":  &types.AttributeValueMemberS{Value: string(indicatorsJSON)},
		},
	})
	return nil
}

// SaveAnalyzedResults는 여러 분석결과를 한 번에 저장합니다
func (r *AnalyzedResultRepositoryImpl) SaveAnalyzedResults(results []*model.AnalyzedResult) error {
	if len(results) == 0 {
		return nil
	}

	// BatchWriteItem 요청을 위한 아이템 생성
	writeRequests := make([]types.WriteRequest, len(results))
	for i, result := range results {
		// SponsorIndicators를 JSON으로 직렬화
		indicatorsJSON, _ := json.Marshal(result.SponsorIndicators)

		writeRequests[i] = types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: map[string]types.AttributeValue{
					"link":               &types.AttributeValueMemberS{Value: result.Link},
					"isSponsored":        &types.AttributeValueMemberBOOL{Value: result.IsSponsored},
					"sponsorProbability": &types.AttributeValueMemberN{Value: strconv.FormatFloat(result.SponsorProbability, 'f', -1, 64)},
					"sponsorIndicators":  &types.AttributeValueMemberS{Value: string(indicatorsJSON)},
				},
			},
		}
	}

	// BatchWriteItem 요청 (DynamoDB는 한 번에 최대 25개 아이템 처리)
	batchSize := 25
	for i := 0; i < len(writeRequests); i += batchSize {
		end := i + batchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		batch := writeRequests[i:end]
		_, err := r.client.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				r.tableName: batch,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
