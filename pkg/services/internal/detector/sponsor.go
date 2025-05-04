package detector

import (
	"fmt"
	"strings"

	repository "github.com/sh5080/ndns-go/pkg/repositories"
	"github.com/sh5080/ndns-go/pkg/services/internal/crawler"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// SponsorService는 스폰서 감지 서비스 인터페이스입니다
type SponsorService interface {
	// DetectSponsor는 블로그 포스트에서 스폰서를 감지합니다
	DetectSponsor(posts []structure.NaverSearchItem) ([]structure.BlogPost, error)
}

// SponsorImpl는 스폰서 감지 서비스 구현체입니다
type SponsorImpl struct {
	crawlerService crawler.CrawlerService
	ocrService     OCRService
	ocrRepo        repository.OCRRepository
}

// NewSponsorService는 새 스폰서 감지 서비스를 생성합니다
func NewSponsorService() SponsorService {
	return &SponsorImpl{
		crawlerService: crawler.NewCrawlerService(),
		ocrService:     NewOCRService(),
		ocrRepo:        repository.NewOCRRepository(),
	}
}

// DetectSponsor는 블로그 포스트에서 스폰서 여부를 감지합니다
func (s *SponsorImpl) DetectSponsor(posts []structure.NaverSearchItem) ([]structure.BlogPost, error) {
	// 1. 네이버 API 텍스트에서 탐지 (빠른 1차 분석)
	results := utils.DetectTextInPosts(posts)

	// 2. 크롤링 및 추가 분석 수행 (유틸리티 함수 사용)
	if s.crawlerService != nil {
		// 크롤러 함수 래핑
		crawlerFunc := func(url string) (*structure.CrawlResult, error) {
			return s.crawlerService.CrawlBlogPost(url)
		}

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

		// 크롤링 및 이미지 분석 수행
		results = utils.AnalyzeWithCrawling(results, crawlerFunc, ocrFunc, ocrCacheFunc)
	}

	return results, nil
}

// detectFromText는 텍스트에서 스폰서 여부를 감지합니다
func (s *SponsorImpl) detectFromText(text string, sponsorType structure.SponsorType) (*structure.SponsorIndicator, float64) {
	// 텍스트 전처리 (소문자 변환)
	textLower := strings.ToLower(text)

	// SPECIAL_CASE_PATTERNS 패턴 확인
	for patternName, pattern := range structure.SPECIAL_CASE_PATTERNS {
		// terms1과 terms2 모두 포함하는지 확인
		term1Found := false
		term2Found := false

		var term1Match, term2Match string

		for _, term1 := range pattern.Terms1 {
			if strings.Contains(textLower, strings.ToLower(term1)) {
				term1Found = true
				term1Match = term1
				break
			}
		}

		for _, term2 := range pattern.Terms2 {
			if strings.Contains(textLower, strings.ToLower(term2)) {
				term2Found = true
				term2Match = term2
				break
			}
		}

		// 두 용어 그룹이 모두 있으면 높은 확률로 판단
		if term1Found && term2Found {
			indicator := &structure.SponsorIndicator{
				Type:        structure.IndicatorTypeKeyword,
				Pattern:     patternName,
				MatchedText: fmt.Sprintf("%s + %s", term1Match, term2Match),
				Probability: 0.9, // 90% 확률
			}

			// 소스 정보 추가
			if sponsorType != "" {
				indicator.Source = &structure.SponsorSource{
					SponsorType: sponsorType,
					Text:        text,
				}
			}

			return indicator, 0.9
		}
	}

	// 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structure.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(textLower, strings.ToLower(exactKeyword)) {
			indicator := &structure.SponsorIndicator{
				Type:        structure.IndicatorTypeExactKeywordRegex,
				Pattern:     exactKeyword,
				MatchedText: exactKeyword,
				Probability: 0.9, // 90% 확률
			}

			// 소스 정보 추가
			if sponsorType != "" {
				indicator.Source = &structure.SponsorSource{
					SponsorType: sponsorType,
					Text:        text,
				}
			}

			return indicator, 0.9
		}
	}

	// 단일 키워드 패턴 확인 (가중치 합산)
	var maxProbability float64 = 0
	var bestMatch string
	var bestPattern string

	for keyword, weight := range structure.SPONSOR_KEYWORDS {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			if weight > maxProbability {
				maxProbability = weight
				bestMatch = keyword
				bestPattern = keyword
			}
		}
	}

	// 가장 높은 가중치의 키워드가 있으면 반환
	if maxProbability > 0 {
		indicator := &structure.SponsorIndicator{
			Type:        structure.IndicatorTypeKeyword,
			Pattern:     bestPattern,
			MatchedText: bestMatch,
			Probability: maxProbability,
		}

		// 소스 정보 추가
		if sponsorType != "" {
			indicator.Source = &structure.SponsorSource{
				SponsorType: sponsorType,
				Text:        text,
			}
		}

		return indicator, maxProbability
	}

	return nil, 0
}
