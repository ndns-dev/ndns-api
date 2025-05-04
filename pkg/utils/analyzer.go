package utils

import (
	"strings"
	"sync"

	"github.com/sh5080/ndns-go/pkg/configs"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

var weightPreference = configs.GetConfig().Weight

// AnalyzeWithCrawling은 블로그 포스트를 크롤링하고 분석합니다
func AnalyzeWithCrawling(posts []structure.BlogPost, crawler CrawlerFunc, ocrExtractor OCRFunc, ocrCache OCRCacheFunc) []structure.BlogPost {
	// 결과 복사 (참조 방지)
	results := make([]structure.BlogPost, len(posts))
	copy(results, posts)

	// 이미 확실한 스폰서로 판단된 포스트가 있는지 확인
	hasDefiniteSponsor := false
	for _, post := range results {
		if post.IsSponsored && post.SponsorProbability >= weightPreference.ExactSponsorKeywords {
			hasDefiniteSponsor = true
			break
		}
	}

	// 확실한 스폰서가 없는 경우에만 추가 크롤링 수행
	if !hasDefiniteSponsor {
		// 동시성 제어를 위한 WaitGroup과 뮤텍스
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, post := range results {
			// 이미 높은 확률의 스폰서로 판단된 경우 크롤링 건너뛰기
			if post.IsSponsored && post.SponsorProbability >= weightPreference.SponsorKeywords {
				continue
			}

			wg.Add(1)

			// 고루틴으로 크롤링 및 분석 (병렬 처리)
			go func(index int, blogPost structure.BlogPost) {
				defer wg.Done()

				// HTML 콘텐츠 크롤링
				crawlResult, err := crawler(blogPost.Link)
				if err != nil {
					// 크롤링 실패 시 현재까지의 결과로 판단
					return
				}

				// 크롤링한 첫 문단 또는 인용구에서 탐지
				if len(crawlResult.FirstParagraph) > 0 {
					isSponsored, probability, indicators := DetectSponsor(crawlResult.FirstParagraph)
					if len(indicators) > 0 {
						// 결과 업데이트 (뮤텍스로 보호)
						mu.Lock()

						// 소스 정보 추가
						for i := range indicators {
							indicators[i].Source = &structure.SponsorSource{
								SponsorType: structure.SponsorTypeFirstParagraph,
								Text:        crawlResult.FirstParagraph,
							}
						}

						// 지표 추가
						results[index].SponsorIndicators = append(results[index].SponsorIndicators, indicators...)

						// 더 높은 확률이 발견되면 업데이트
						if probability > results[index].SponsorProbability {
							results[index].SponsorProbability = probability

							// 스폰서 판단 업데이트
							results[index].IsSponsored = isSponsored
						}
						mu.Unlock()
					}
				}

				// 이미지 및 스티커 분석
				analyzeImages(crawlResult, ocrExtractor, ocrCache, &results[index], &mu)

			}(i, post)
		}

		// 모든 고루틴이 완료될 때까지 대기
		wg.Wait()
	}

	return results
}

// analyzeImages는 이미지들을 OCR로 분석합니다
func analyzeImages(crawlResult *structure.CrawlResult, ocrExtractor OCRFunc, ocrCache OCRCacheFunc, post *structure.BlogPost, mu *sync.Mutex) {
	// 첫 번째 스티커 이미지 URL 탐지
	if crawlResult.StickerURL != "" {
		analyzeImage(crawlResult.StickerURL, structure.SponsorTypeSticker, ocrExtractor, ocrCache, post, mu)
	}

	// 첫 번째 일반 이미지 URL 탐지
	if crawlResult.ImageURL != "" {
		analyzeImage(crawlResult.ImageURL, structure.SponsorTypeImage, ocrExtractor, ocrCache, post, mu)
	}
}

// analyzeImage는 단일 이미지를 OCR로 분석합니다
func analyzeImage(imageURL string, sponsorType structure.SponsorType, ocrExtractor OCRFunc, ocrCache OCRCacheFunc, post *structure.BlogPost, mu *sync.Mutex) {
	// DB에서 OCR 결과 조회
	ocrResult, found := ocrCache(imageURL)

	var ocrText string
	if found && ocrResult != "" {
		// 캐시에서 OCR 결과 가져오기
		ocrText = ocrResult
	} else {
		// OCR 실행
		var err error
		ocrText, err = ocrExtractor(imageURL)
		if err != nil || ocrText == "" {
			return
		}
	}

	// OCR 텍스트에서 스폰서 감지
	isSponsored, probability, indicators := DetectSponsorInOCR(ocrText)
	if len(indicators) > 0 {
		// 결과 업데이트 (뮤텍스로 보호)
		mu.Lock()

		// 소스 정보 추가
		for i := range indicators {
			indicators[i].Source = &structure.SponsorSource{
				SponsorType: sponsorType,
				Text:        ocrText,
			}
		}

		// 지표 추가
		post.SponsorIndicators = append(post.SponsorIndicators, indicators...)

		// 더 높은 확률이 발견되면 업데이트
		if probability > post.SponsorProbability {
			post.SponsorProbability = probability

			// 스폰서 판단 업데이트
			post.IsSponsored = isSponsored
		}
		mu.Unlock()
	}
}

// DetectSponsorInOCR은 OCR 텍스트에서 스폰서 여부를 감지합니다
func DetectSponsorInOCR(ocrText string) (bool, float64, []structure.SponsorIndicator) {
	var indicators []structure.SponsorIndicator
	maxProbability := 0.0
	isSponsored := false

	// 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structure.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(strings.ToLower(ocrText), strings.ToLower(exactKeyword)) {
			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeExactKeywordRegex,
				Pattern:     exactKeyword,
				MatchedText: exactKeyword,
				Probability: weightPreference.ExactSponsorKeywords,
			}

			indicators = append(indicators, indicator)
			maxProbability = weightPreference.ExactSponsorKeywords
			isSponsored = true

			// 높은 확률이면 바로 반환
			return isSponsored, maxProbability, indicators
		}
	}

	// 단일 키워드 패턴 확인
	for keyword, weight := range structure.SPONSOR_KEYWORDS {
		if weight >= weightPreference.LowSponsorKeywords && strings.Contains(strings.ToLower(ocrText), strings.ToLower(keyword)) {
			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeKeyword,
				Pattern:     keyword,
				MatchedText: keyword,
				Probability: weight,
			}

			indicators = append(indicators, indicator)

			if weight > maxProbability {
				maxProbability = weight
			}

			// 확률이 70% 이상이면 스폰서로 판단
			if weight >= weightPreference.SponsorKeywords {
				isSponsored = true
			}
		}
	}

	return isSponsored, maxProbability, indicators
}

// 필요한 함수 타입 정의
type CrawlerFunc func(url string) (*structure.CrawlResult, error)
type OCRFunc func(imageURL string) (string, error)
type OCRCacheFunc func(imageURL string) (string, bool)
