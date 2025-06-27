package api

import (
	"fmt"

	naver "github.com/sh5080/ndns-go/pkg/clients"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/analyzer"
	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
	"github.com/sh5080/ndns-go/pkg/services/internal/queue"
	request "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// SearchImpl는 검색 서비스 구현체입니다
type SearchImpl struct {
	_interface.Service
	naverClient *naver.NaverAPIClient
	postService _interface.PostService
}

// NewSearchService는 새 검색 서비스를 생성합니다
func NewSearchService() _interface.SearchService {
	config := configs.GetConfig()
	naverClient := naver.NewNaverAPIClient(config)
	queueService := queue.NewSqsService()
	analyzerService := analyzer.NewAnalyzerService()
	ocrService := detector.NewOcrService(queueService, analyzerService)
	postService := detector.NewPostService(ocrService)

	return &SearchImpl{
		Service:     _interface.Service{Config: config},
		naverClient: naverClient,
		postService: postService,
	}
}

// SearchAnalyzedResponses는 검색어로 블로그 포스트를 검색합니다
func (s *SearchImpl) SearchAnalyzedResponses(req request.SearchQuery) ([]structure.AnalyzedResponse, int, error) {
	if s.naverClient == nil {
		return nil, 0, fmt.Errorf("네이버 API 클라이언트가 초기화되지 않았습니다")
	}

	// 네이버 블로그 검색 API 호출
	searchResp, err := s.naverClient.SearchBlog(req.Query, req.Limit, req.Offset+1)
	if err != nil {
		return nil, 0, fmt.Errorf("네이버 블로그 검색 실패: %v", err)
	}

	// 스폰서 감지 (실패해도 계속 진행)
	var posts []structure.AnalyzedResponse
	posts, err = s.postService.DetectPosts(searchResp.Items)
	if err != nil {
		fmt.Printf("스폰서 감지 중 무시된 오류: %v\n", err)
		// 오류 발생 시 빈 슬라이스 반환
		posts = []structure.AnalyzedResponse{}
	}

	// 네이버 API에서 반환한 총 결과 수 반환
	return posts, searchResp.Total, nil
}
