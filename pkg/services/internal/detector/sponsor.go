package detector

import (
	"fmt"

	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	"github.com/sh5080/ndns-go/pkg/services/internal/analyzer"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// SponsorImpl는 스폰서 감지 서비스 구현체입니다
type SponsorImpl struct {
	_interface.Service
	ocrService _interface.OCRService
	ocrRepo    _interface.OCRRepository
}

// NewSponsorService는 새 스폰서 감지 서비스를 생성합니다
func NewSponsorService() _interface.SponsorService {
	return &SponsorImpl{
		Service:    _interface.Service{},
		ocrService: NewOCRService(),
		ocrRepo:    repository.NewOCRRepository(),
	}
}

// DetectSponsor는 블로그 포스트에서 스폰서 여부를 감지합니다
func (s *SponsorImpl) DetectSponsor(posts []structure.NaverSearchItem) ([]structure.BlogPost, error) {
	// OCR 함수 래핑
	ocrFunc := func(imageURL string) (string, error) {
		if s.ocrService != nil {
			return s.ocrService.ExtractTextFromImage(imageURL)
		}
		return "", fmt.Errorf("OCR 서비스가 초기화되지 않음")
	}

	// OCR 캐시 함수 래핑
	ocrCacheFunc := func(imageURL string) (string, bool) {
		if s.ocrRepo != nil {
			cache, err := s.ocrRepo.GetOCRCache(imageURL)
			if err == nil && cache != nil {
				return cache.TextDetected, true
			}
		}
		return "", false
	}
	// 1. 네이버 API 텍스트에서 탐지 (빠른 1차 분석)
	results := DetectTextInPosts(posts, ocrFunc, ocrCacheFunc)

	// 크롤링 및 이미지 분석 수행
	results = analyzer.Crawl(results, ocrFunc, ocrCacheFunc)

	return results, nil
}
