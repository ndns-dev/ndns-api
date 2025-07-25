package _interface

import (
	"net/http"

	"github.com/sh5080/ndns-go/pkg/configs"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

type Service struct {
	Config *configs.EnvConfig
	Client *http.Client
}

// ServiceContainer는 모든 서비스 인스턴스를 보관합니다
type ServiceContainer struct {
	DetectorService DetectorService
	SearchService   SearchService
	AnalyzerService AnalyzerService
	OcrRepository   OcrRepository
}

// SearchService는 검색 서비스 인터페이스입니다
type SearchService interface {
	// SearchAnalyzedResponses는 검색어로 블로그 포스트를 검색합니다
	SearchAnalyzedResponses(req requestDto.SearchQuery, reqId string) ([]responseDto.AnalyzedResponse, int, error)
	// GetJobDetail은 작업 상세 정보를 조회합니다
	GetJobDetail(id string) (model.OcrQueueState, model.OcrResult, error)
}

type AnalyzerService interface {
	// AnalyzeText는 텍스트를 분석하고 협찬 여부를 판단합니다
	AnalyzeText(text string) (*responseDto.AnalyzedResponse, error)
	// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정합니다
	AnalyzeCycle(state model.OcrQueueState, result model.OcrResult) (*responseDto.AnalyzeJobResponse, error)
	// AnalyzePosts는 블로그 포스트에서 협찬 관련 텍스트를 감지합니다
	AnalyzePosts(posts []structure.NaverSearchItem, reqId string) ([]responseDto.AnalyzedResponse, error)
	// GetExistingPosts는 기존 분석결과를 조회합니다
	GetExistingPosts(posts []structure.NaverSearchItem) ([]responseDto.AnalyzedResponse, error)
}

// CrawlerService는 블로그 콘텐츠를 크롤링하는 인터페이스입니다
type CrawlerService interface {
	// CrawlAnalyzedResponse는 블로그 포스트 Url에서 콘텐츠를 크롤링합니다
	CrawlAnalyzedResponse(url string) (*structure.CrawlResult, error)
}

// DetectorService는 탐지 처리를 관리하는 인터페이스입니다
type DetectorService interface {
	// RequestNextOcr은 다음 탐지 처리를 요청합니다
	RequestNextOcr(state model.OcrQueueState) error
}

// QueueService는 큐 작업을 처리하는 인터페이스입니다
type QueueService interface {
	// SendQueue는 큐 작업을 SQS에 전송합니다
	SendQueue(queueState model.OcrQueueState) error
}
