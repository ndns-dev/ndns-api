package api

import (
	"fmt"

	naver "github.com/sh5080/ndns-go/pkg/clients"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// SearchImpl는 검색 서비스 구현체입니다
type SearchImpl struct {
	_interface.Service
	naverClient     *naver.NaverAPIClient
	analyzerService _interface.AnalyzerService
	ocrRepository   _interface.OcrRepository
}

// NewSearchService는 새 검색 서비스를 생성합니다
func NewSearchService(analyzerService _interface.AnalyzerService, ocrRepository _interface.OcrRepository) _interface.SearchService {
	config := configs.GetConfig()
	naverClient := naver.NewNaverAPIClient(config)

	return &SearchImpl{
		Service:         _interface.Service{Config: config},
		naverClient:     naverClient,
		analyzerService: analyzerService,
		ocrRepository:   ocrRepository,
	}
}

// SearchAnalyzedResponses는 검색어로 블로그 포스트를 검색합니다
func (s *SearchImpl) SearchAnalyzedResponses(req requestDto.SearchQuery, reqId string) ([]responseDto.AnalyzedResponse, int, error) {
	if s.naverClient == nil {
		return nil, 0, fmt.Errorf("네이버 API 클라이언트가 초기화되지 않았습니다")
	}

	// 네이버 블로그 검색 API 호출
	searchResp, err := s.naverClient.SearchBlog(req.Query, req.Limit, req.Offset+1)
	if err != nil {
		return nil, 0, fmt.Errorf("네이버 블로그 검색 실패: %v", err)
	}

	// 스폰서 감지 (실패해도 계속 진행)
	var posts []responseDto.AnalyzedResponse
	posts, err = s.analyzerService.AnalyzePosts(searchResp.Items, reqId)
	if err != nil {
		fmt.Printf("스폰서 감지 중 무시된 오류: %v\n", err)
		// 오류 발생 시 빈 슬라이스 반환
		posts = []responseDto.AnalyzedResponse{}
	}

	// 네이버 API에서 반환한 총 결과 수 반환
	return posts, searchResp.Total, nil
}

// 현재 진행중인 작업 조회
func (s *SearchImpl) GetJobDetail(id string) (model.OcrQueueState, model.OcrResult, error) {

	job, err := s.ocrRepository.GetOcrJob(id)

	if err != nil {
		return model.OcrQueueState{}, model.OcrResult{}, err
	}

	result, err := s.ocrRepository.GetOcrResult(id)
	if err != nil {
		return model.OcrQueueState{}, model.OcrResult{}, err
	}

	return *job, *result, nil
}
