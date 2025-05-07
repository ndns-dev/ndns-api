package detector

import (
	"fmt"
	"strings"
	"sync"

	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/crawler"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// DetectTextInPosts는 여러 포스트에서 동시에 스폰서 관련 텍스트를 탐지합니다
func DetectTextInPosts(posts []structure.NaverSearchItem, ocrExtractor _interface.OCRFunc, ocrCache _interface.OCRCacheFunc) []structure.BlogPost {
	// 결과를 저장할 슬라이스 초기화
	results := make([]structure.BlogPost, len(posts))

	// 동시성 제어를 위한 WaitGroup
	var wg sync.WaitGroup

	// 동시성 제어를 위한 뮤텍스와 채널
	var mu sync.Mutex
	doneCh := make(chan struct{})

	// 각 포스트에 대해 병렬로 처리
	for i, post := range posts {
		wg.Add(1)

		// 고루틴으로 포스트 분석
		go func(index int, item structure.NaverSearchItem) {
			defer wg.Done()

			// 외부 신호 확인 (다른 고루틴에서 이미 확실한 스폰서를 발견한 경우)
			select {
			case <-doneCh:
				// 다른 고루틴에서 이미 확실한 스폰서를 발견했으므로 종료
				return
			default:
				// 계속 진행
			}

			// 블로그 포스트 초기화
			blogPost := structure.BlogPost{
				NaverSearchItem:    item,
				IsSponsored:        false,
				SponsorProbability: 0,
				SponsorIndicators:  []structure.SponsorIndicator{},
			}

			// 1. Description 텍스트 탐지 수행
			isSponsored, probability, indicators := DetectSponsor(item.Description, structure.SponsorTypeDescription)

			if isSponsored {
				blogPost.IsSponsored = isSponsored
				blogPost.SponsorProbability = probability
				blogPost.SponsorIndicators = indicators
			} else {
				// 2. Description에서 스폰서 탐지 실패시 본문 크롤링
				crawlResult, err := crawler.CrawlBlogPost(item.Link)
				if err != nil {
					fmt.Printf("%s", err.Error())
				}
				isSponsored, probability, indicators = DetectSponsor(crawlResult.FirstParagraph, structure.SponsorTypeFirstParagraph)
				if isSponsored {
					blogPost.IsSponsored = isSponsored
					blogPost.SponsorProbability = probability
					blogPost.SponsorIndicators = indicators
				} else {
					if crawlResult.StickerURL != "" {
						// 3. 크롤링한 본문에서 스폰서 탐지 실패시 스티커 이미지 OCR
						ocrText, err := ocrExtractor(crawlResult.StickerURL)
						if err != nil {
							fmt.Printf("%s", err.Error())
						}
						isSponsored, probability, indicators = DetectSponsor(ocrText, structure.SponsorTypeSticker)
						if isSponsored {
							blogPost.IsSponsored = isSponsored
							blogPost.SponsorProbability = probability
							blogPost.SponsorIndicators = indicators
						}
					}

					// 4. 스티커 URL이 없거나 스폰서 탐지 실패시 일반 이미지 OCR
					if crawlResult.ImageURL != "" {
						ocrText, err := ocrExtractor(crawlResult.ImageURL)
						if err != nil {
							fmt.Printf("%s", err.Error())
						}
						isSponsored, probability, indicators = DetectSponsor(ocrText, structure.SponsorTypeImage)
						if isSponsored {
							blogPost.IsSponsored = isSponsored
							blogPost.SponsorProbability = probability
							blogPost.SponsorIndicators = indicators
						}
					}
				}
			}

			// 확률이 90% 이상이면 확실한 스폰서로 판단하고 다른 고루틴에게 알림
			if isSponsored && probability >= 0.9 {
				// 뮤텍스로 경쟁 상태 방지
				mu.Lock()
				select {
				case <-doneCh:
					// 이미 닫힌 경우 무시
				default:
					// 채널을 닫아 다른 고루틴에게 알림
					close(doneCh)
				}
				mu.Unlock()
			}

			// 결과 저장
			mu.Lock()
			results[index] = blogPost
			mu.Unlock()
		}(i, post)
	}

	// 모든 고루틴이 완료될 때까지 대기
	wg.Wait()

	return results
}

// DetectSponsor는 텍스트에서 스폰서 여부를 감지합니다
func DetectSponsor(text string, sourceType structure.SponsorType) (bool, float64, []structure.SponsorIndicator) {
	var indicators []structure.SponsorIndicator
	maxProbability := 0.0
	isSponsored := false
	text = strings.ReplaceAll(text, " ", "")
	// 1. SPECIAL_CASE_PATTERNS 패턴 확인
	for _, pattern := range structure.SPECIAL_CASE_PATTERNS {
		// terms1과 terms2 모두 포함하는지 확인
		term1Found := false
		term2Found := false

		var term1Match, term2Match string

		if strings.Contains(text, pattern.Terms1) {
			term1Found = true
			term1Match = pattern.Terms1
		}

		for _, term2 := range pattern.Terms2 {
			if strings.Contains(text, term2) {
				term2Found = true
				term2Match = term2
				break
			}
		}
		// 두 용어 그룹이 모두 있으면 높은 확률로 판단
		if term1Found && term2Found {
			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeKeyword,
				Pattern:     structure.PatternTypeSpecial,
				MatchedText: fmt.Sprintf("%s, %s", term1Match, term2Match),
				Probability: 0.9, // 90% 확률
				Source: structure.SponsorSource{
					SponsorType: sourceType,
					Text:        text,
				},
			}

			indicators = append(indicators, indicator)
			maxProbability = 0.9
			isSponsored = true

			return isSponsored, maxProbability, indicators
		}
	}

	// 2. 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structure.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(text, exactKeyword) {
			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeExactKeywordRegex,
				Pattern:     structure.PatternTypeExact,
				MatchedText: exactKeyword,
				Probability: 0.9, // 90% 확률
				Source: structure.SponsorSource{
					SponsorType: sourceType,
					Text:        text,
				},
			}

			indicators = append(indicators, indicator)
			maxProbability = 0.9
			isSponsored = true

			// 높은 확률이면 바로 반환
			return isSponsored, maxProbability, indicators
		}
	}

	// 3. 단일 키워드 패턴 확인 (가중치 합산)
	for keyword, weight := range structure.SPONSOR_KEYWORDS {
		if strings.Contains(text, keyword) {
			if weight > maxProbability {
				maxProbability = weight

				indicator := structure.SponsorIndicator{
					Type:        structure.IndicatorTypeKeyword,
					Pattern:     structure.PatternTypeNormal,
					MatchedText: text,
					Probability: weight,
					Source: structure.SponsorSource{
						SponsorType: sourceType,
						Text:        text,
					},
				}

				// 지표 추가
				indicators = append(indicators, indicator)

				// 확률이 70% 이상이면 스폰서로 판단
				if weight >= 0.7 {
					isSponsored = true
				}
			}
		}
	}

	return isSponsored, maxProbability, indicators
}

// detectText는 텍스트에서 스폰서 여부를 탐지합니다
func detectText(text string) (*structure.SponsorIndicator, float64) {
	// SPECIAL_CASE_PATTERNS 패턴 확인
	for _, pattern := range structure.SPECIAL_CASE_PATTERNS {
		// terms1과 terms2 모두 포함하는지 확인
		term1Found := false
		term2Found := false

		var term1Match, term2Match string

		if strings.Contains(text, pattern.Terms1) {
			term1Found = true
			term1Match = pattern.Terms1
		}

		for _, term2 := range pattern.Terms2 {
			if strings.Contains(text, term2) {
				term2Found = true
				term2Match = term2
				break
			}
		}

		// 두 용어 그룹이 모두 있으면 높은 확률로 판단
		if term1Found && term2Found {
			return &structure.SponsorIndicator{
				Type:        structure.IndicatorTypeKeyword,
				Pattern:     structure.PatternTypeSpecial,
				MatchedText: fmt.Sprintf("%s, %s", term1Match, term2Match),
				Probability: 0.9, // 90% 확률
			}, 0.9
		}
	}

	// 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structure.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(text, exactKeyword) {
			return &structure.SponsorIndicator{
				Type:        structure.IndicatorTypeExactKeywordRegex,
				Pattern:     structure.PatternTypeExact,
				MatchedText: exactKeyword,
				Probability: 0.9, // 90% 확률
			}, 0.9
		}
	}

	// 단일 키워드 패턴 확인 (가중치 합산)
	var maxProbability float64 = 0
	var bestMatch string

	for keyword, weight := range structure.SPONSOR_KEYWORDS {
		if strings.Contains(text, keyword) {
			if weight > maxProbability {
				maxProbability = weight
				bestMatch = keyword
			}
		}
	}

	// 가장 높은 가중치의 키워드가 있으면 반환
	if maxProbability > 0 {
		return &structure.SponsorIndicator{
			Type:        structure.IndicatorTypeKeyword,
			Pattern:     structure.PatternTypeNormal,
			MatchedText: bestMatch,
			Probability: maxProbability,
		}, maxProbability
	}

	return nil, 0
}

// DetectTextAsync는 비동기로 텍스트에서 스폰서 여부를 탐지합니다 (채널 기반)
func DetectTextAsync(texts []string) <-chan SponsorDetectionResult {
	resultCh := make(chan SponsorDetectionResult)

	go func() {
		// 모든 텍스트 처리 후 채널 닫기
		defer close(resultCh)

		// 동시성 제어를 위한 WaitGroup
		var wg sync.WaitGroup

		// 채널 접근을 위한 뮤텍스
		var mu sync.Mutex

		// 각 텍스트에 대해 병렬로 처리
		for i, text := range texts {
			wg.Add(1)

			go func(index int, content string) {
				defer wg.Done()

				indicator, probability := detectText(content)

				// 뮤텍스로 채널 접근 보호
				mu.Lock()
				resultCh <- SponsorDetectionResult{
					Index:       index,
					Indicator:   indicator,
					Probability: probability,
				}
				mu.Unlock()
			}(i, text)
		}

		// 모든 고루틴이 완료될 때까지 대기
		wg.Wait()
	}()

	return resultCh
}

// SponsorDetectionResult는 스폰서 감지 결과를 나타냅니다
type SponsorDetectionResult struct {
	Index       int
	Indicator   *structure.SponsorIndicator
	Probability float64
}
