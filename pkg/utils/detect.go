package utils

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sh5080/ndns-go/pkg/types/structures"
)

// DetectTextInPosts는 여러 포스트에서 동시에 스폰서 관련 텍스트를 탐지합니다
func DetectTextInPosts(posts []structures.NaverSearchItem) []structures.BlogPost {
	// 결과를 저장할 슬라이스 초기화
	results := make([]structures.BlogPost, len(posts))

	// 동시성 제어를 위한 WaitGroup
	var wg sync.WaitGroup

	// 동시성 제어를 위한 뮤텍스와 채널
	var mu sync.Mutex
	doneCh := make(chan struct{})

	// 각 포스트에 대해 병렬로 처리
	for i, post := range posts {
		wg.Add(1)

		// 고루틴으로 포스트 분석
		go func(index int, item structures.NaverSearchItem) {
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
			blogPost := structures.BlogPost{
				NaverSearchItem:    item,
				IsSponsored:        false,
				SponsorProbability: 0,
				SponsorIndicators:  []structures.SponsorIndicator{},
			}

			// 텍스트 탐지 수행
			isSponsored, probability, indicators := DetectSponsor(item.Description)
			blogPost.IsSponsored = isSponsored
			blogPost.SponsorProbability = probability
			blogPost.SponsorIndicators = indicators

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
func DetectSponsor(text string) (bool, float64, []structures.SponsorIndicator) {
	var indicators []structures.SponsorIndicator
	maxProbability := 0.0
	isSponsored := false

	// 텍스트 전처리 (소문자 변환)
	textLower := strings.ToLower(text)

	// SPECIAL_CASE_PATTERNS 패턴 확인
	for patternName, pattern := range structures.SPECIAL_CASE_PATTERNS {
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
			indicator := structures.SponsorIndicator{
				Type:        structures.IndicatorTypeKeyword,
				Pattern:     patternName,
				MatchedText: fmt.Sprintf("%s + %s", term1Match, term2Match),
				Probability: 0.9, // 90% 확률
			}

			indicators = append(indicators, indicator)
			maxProbability = 0.9
			isSponsored = true

			// 높은 확률이면 바로 반환
			return isSponsored, maxProbability, indicators
		}
	}

	// 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structures.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(textLower, strings.ToLower(exactKeyword)) {
			indicator := structures.SponsorIndicator{
				Type:        structures.IndicatorTypeExactKeywordRegex,
				Pattern:     exactKeyword,
				MatchedText: exactKeyword,
				Probability: 0.9, // 90% 확률
			}

			indicators = append(indicators, indicator)
			maxProbability = 0.9
			isSponsored = true

			// 높은 확률이면 바로 반환
			return isSponsored, maxProbability, indicators
		}
	}

	// 단일 키워드 패턴 확인 (가중치 합산)
	for keyword, weight := range structures.SPONSOR_KEYWORDS {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			if weight > maxProbability {
				maxProbability = weight

				indicator := structures.SponsorIndicator{
					Type:        structures.IndicatorTypeKeyword,
					Pattern:     keyword,
					MatchedText: keyword,
					Probability: weight,
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
func detectText(text string) (*structures.SponsorIndicator, float64) {
	// 텍스트 전처리 (소문자 변환)
	textLower := strings.ToLower(text)

	// SPECIAL_CASE_PATTERNS 패턴 확인
	for patternName, pattern := range structures.SPECIAL_CASE_PATTERNS {
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
			return &structures.SponsorIndicator{
				Type:        structures.IndicatorTypeKeyword,
				Pattern:     patternName,
				MatchedText: term1Match + " + " + term2Match,
				Probability: 0.9, // 90% 확률
			}, 0.9
		}
	}

	// 정확한 스폰서 키워드 확인
	for _, exactKeyword := range structures.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(textLower, strings.ToLower(exactKeyword)) {
			return &structures.SponsorIndicator{
				Type:        structures.IndicatorTypeExactKeywordRegex,
				Pattern:     exactKeyword,
				MatchedText: exactKeyword,
				Probability: 0.9, // 90% 확률
			}, 0.9
		}
	}

	// 단일 키워드 패턴 확인 (가중치 합산)
	var maxProbability float64 = 0
	var bestMatch string
	var bestPattern string

	for keyword, weight := range structures.SPONSOR_KEYWORDS {
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
		return &structures.SponsorIndicator{
			Type:        structures.IndicatorTypeKeyword,
			Pattern:     bestPattern,
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
	Indicator   *structures.SponsorIndicator
	Probability float64
}
