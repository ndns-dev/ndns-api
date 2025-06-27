package common

import (
	"fmt"
	"strings"

	constants "github.com/sh5080/ndns-go/pkg/types"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	utils "github.com/sh5080/ndns-go/pkg/utils"
)

// DetectSponsor는 텍스트에서 협찬 여부를 감지합니다
func DetectSponsor(text string, sourceType structure.SponsorType) (bool, float64, []structure.SponsorIndicator) {
	var indicators []structure.SponsorIndicator
	maxProbability := 0.0
	isSponsored := false
	text = strings.ReplaceAll(text, " ", "")
	utils.DebugLog("협찬 탐지 시작: %s\n", text)
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
				Probability: structure.Accuracy.Exact,
				Source: structure.SponsorSource{
					SponsorType: sourceType,
					Text:        text,
				},
			}

			indicators = append(indicators, indicator)
			maxProbability = structure.Accuracy.Exact
			isSponsored = true

			return isSponsored, maxProbability, indicators
		}
	}

	// 2. 정확한 협찬 키워드 확인
	for _, exactKeyword := range structure.EXACT_SPONSOR_KEYWORDS_PATTERNS {
		if strings.Contains(text, exactKeyword) {
			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeExactKeywordRegex,
				Pattern:     structure.PatternTypeExact,
				MatchedText: exactKeyword,
				Probability: structure.Accuracy.Exact,
				Source: structure.SponsorSource{
					SponsorType: sourceType,
					Text:        text,
				},
			}

			indicators = append(indicators, indicator)
			maxProbability = structure.Accuracy.Exact
			isSponsored = true

			// 높은 확률이면 바로 반환
			return isSponsored, maxProbability, indicators
		}
	}

	// 3. 단일 키워드 패턴 확인 (가중치 합산)
	totalWeight := 0.0
	for keyword, weight := range structure.SPONSOR_KEYWORDS {
		if strings.Contains(text, keyword) {
			// 가중치 합산
			totalWeight += weight

			indicator := structure.SponsorIndicator{
				Type:        structure.IndicatorTypeKeyword,
				Pattern:     structure.PatternTypeNormal,
				MatchedText: keyword,
				Probability: weight,
				Source: structure.SponsorSource{
					SponsorType: sourceType,
					Text:        text,
				},
			}

			// 지표 추가
			indicators = append(indicators, indicator)
		}
	}

	// 합산된 가중치가 Possible 초과하면 스폰서로 판단
	if totalWeight > structure.Accuracy.Possible {
		isSponsored = true
	}

	return isSponsored, totalWeight, indicators
}

// IsSponsorImageUrl은 이미지 URL이 협찬 도메인 패턴과 일치하는지 확인합니다
func IsSponsorImageUrl(imageUrl string) string {
	if imageUrl == "" {
		return ""
	}

	for _, domain := range constants.SPONSOR_DOMAINS {
		if strings.Contains(strings.ToLower(imageUrl), strings.ToLower(domain)) {
			return domain
		}
	}
	return ""
}

// CheckSponsorImagesInCrawlResult는 크롤링 결과의 모든 이미지 URL을 검사합니다
func CheckSponsorImagesInCrawlResult(result *structure.CrawlResult) (bool, float64, []structure.SponsorIndicator) {
	if result == nil {
		return false, 0, nil
	}

	urls := []string{
		result.FirstImageUrl,
		result.LastImageUrl,
		result.FirstStickerUrl,
		result.SecondStickerUrl,
		result.LastStickerUrl,
	}

	for _, url := range urls {
		domain := IsSponsorImageUrl(url)
		if domain != "" {
			indicators := []structure.SponsorIndicator{
				{
					Type:        structure.IndicatorTypeExactKeywordRegex,
					Pattern:     structure.PatternTypeExact,
					MatchedText: url,
					Probability: 1.0,
					Source: structure.SponsorSource{
						SponsorType: structure.SponsorTypeDomain,
						Text:        domain,
					},
				},
			}
			return true, 1.0, indicators
		}
	}

	return false, 0, nil
}
