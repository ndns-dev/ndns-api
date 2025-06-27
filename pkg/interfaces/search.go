package _interface

import (
	request "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// SearchService는 검색 서비스 인터페이스입니다
type SearchService interface {
	// SearchAnalyzedResponses는 검색어로 블로그 포스트를 검색합니다
	SearchAnalyzedResponses(req request.SearchQuery) ([]structure.AnalyzedResponse, int, error)
}

// PostService는 포스트 감지 서비스 인터페이스입니다
type PostService interface {
	// DetectPosts는 블로그 포스트에서 협찬 관련 텍스트를 감지합니다
	DetectPosts(posts []structure.NaverSearchItem) ([]structure.AnalyzedResponse, error)
}

type AnalyzerService interface {
	// AnalyzeText는 텍스트를 분석하고 협찬 여부를 판단합니다
	AnalyzeText(req request.AnalyzeTextParam) (*structure.AnalyzedResponse, error)
	// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정합니다
	AnalyzeCycle(state model.OcrQueueState, result model.OcrResult) (*structure.AnalyzedResponse, error)
}

// CrawlerService는 블로그 콘텐츠를 크롤링하는 인터페이스입니다
type CrawlerService interface {
	// CrawlAnalyzedResponse는 블로그 포스트 Url에서 콘텐츠를 크롤링합니다
	CrawlAnalyzedResponse(url string) (*structure.CrawlResult, error)
}
