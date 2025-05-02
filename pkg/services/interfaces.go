package service

import (
	"net/http"

	"github.com/sh5080/ndns-go/pkg/configs"
	"github.com/sh5080/ndns-go/pkg/types/structures"
)

// searchService는 검색 서비스 구현체입니다
type Service struct {
	config *configs.EnvConfig
	client *http.Client
}

// SearchService는 검색 서비스 인터페이스입니다
type SearchService interface {
	// SearchBlogPosts는 검색어로 블로그 포스트를 검색합니다
	SearchBlogPosts(query string, count int, start int) ([]structures.BlogPost, error)
}

// SponsorDetectorService는 스폰서 감지 서비스 인터페이스입니다
type SponsorDetectorService interface {
	// DetectSponsor는 블로그 포스트에서 스폰서를 감지합니다
	DetectSponsor(posts []structures.NaverSearchItem) ([]structures.BlogPost, error)
}

// SearchImpl는 검색 서비스 구현체입니다
type SearchImpl struct {
	Service
	crawlerService CrawlerService
	ocrService     OCRService
	dbService      DBService
}

// CrawlerService는 블로그 콘텐츠를 크롤링하는 인터페이스입니다
type CrawlerService interface {
	// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
	CrawlBlogPost(url string) (*structures.CrawlResult, error)
}

// OCRService는 이미지에서 텍스트를 추출하는 인터페이스입니다
type OCRService interface {
	// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
	ExtractTextFromImage(imageURL string) (string, error)
}

// DBService는 데이터베이스 작업을 수행하는 인터페이스입니다
type DBService interface {
	// GetOCRCache는 이미지 URL에 대한 OCR 캐시를 가져옵니다
	GetOCRCache(imageURL string) (*structures.OCRCache, error)

	// SaveOCRCache는 이미지 URL에 대한 OCR 결과를 저장합니다
	SaveOCRCache(imageURL string, textDetected string, imageType string) error
}
