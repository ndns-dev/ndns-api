package analyzer

import (
	"strings"

	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// CreateAnalyzedResponse는 기본 블로그 포스트 구조체를 생성합니다
func CreateAnalyzedResponse(item structure.NaverSearchItem) structure.AnalyzedResponse {
	return structure.AnalyzedResponse{
		NaverSearchItem:    item,
		IsSponsored:        false,
		SponsorProbability: 0,
		SponsorIndicators:  []structure.SponsorIndicator{},
	}
}

// CreateSponsoredAnalyzedResponse는 스폰서된 블로그 포스트 구조체를 생성합니다
func CreateSponsoredAnalyzedResponse(
	item structure.NaverSearchItem,
	probability float64,
	matchedText string,
	indicatorType structure.IndicatorType,
	patternType structure.PatternType,
	sponsorType structure.SponsorType,
	sourceText string,
) structure.AnalyzedResponse {
	// 협찬 표시자 생성
	indicator := structure.SponsorIndicator{
		Type:        indicatorType,
		Pattern:     patternType,
		MatchedText: matchedText,
		Probability: probability,
		Source: structure.SponsorSource{
			SponsorType: sponsorType,
			Text:        sourceText,
		},
	}

	// 블로그 포스트 생성
	return structure.AnalyzedResponse{
		NaverSearchItem:    item,
		IsSponsored:        true,
		SponsorProbability: probability,
		SponsorIndicators:  []structure.SponsorIndicator{indicator},
		Error:              "",
	}
}

// AddIndicator는 블로그 포스트에 협찬 표시자를 추가합니다
func AddIndicator(
	post *structure.AnalyzedResponse,
	indicatorType structure.IndicatorType,
	patternType structure.PatternType,
	matchedText string,
	probability float64,
	sponsorType structure.SponsorType,
	sourceText string,
) {
	// 새 협찬 표시자 생성
	indicator := structure.SponsorIndicator{
		Type:        indicatorType,
		Pattern:     patternType,
		MatchedText: matchedText,
		Probability: probability,
		Source: structure.SponsorSource{
			SponsorType: sponsorType,
			Text:        sourceText,
		},
	}

	// 기존 표시자에 추가
	post.SponsorIndicators = append(post.SponsorIndicators, indicator)

	// 확률 업데이트 (가장 높은 확률 유지)
	if probability > post.SponsorProbability {
		post.SponsorProbability = probability
	}

	// 확률이 0보다 크면 스폰서로 표시
	if probability > 0 {
		post.IsSponsored = true
	}
}

// CreateSponsorIndicator는 협찬 표시자를 생성합니다
func CreateSponsorIndicator(
	indicatorType structure.IndicatorType,
	patternType structure.PatternType,
	matchedText string,
	probability float64,
	sponsorType structure.SponsorType,
	sourceText string,
) structure.SponsorIndicator {
	return structure.SponsorIndicator{
		Type:        indicatorType,
		Pattern:     patternType,
		MatchedText: matchedText,
		Probability: probability,
		Source: structure.SponsorSource{
			SponsorType: sponsorType,
			Text:        sourceText,
		},
	}
}

// UpdateAnalyzedResponseWithSponsorInfo는 협찬 감지 결과를 블로그 포스트에 업데이트합니다
func UpdateAnalyzedResponseWithSponsorInfo(
	blogPost *structure.AnalyzedResponse,
	isSponsored bool,
	probability float64,
	indicators []structure.SponsorIndicator,
	errorMessage ...string,
) {
	if !isSponsored {
		// 에러 메시지가 제공된 경우 설정
		if len(errorMessage) > 0 && errorMessage[0] != "" {
			blogPost.Error = errorMessage[0]
		}
		return
	}

	blogPost.IsSponsored = isSponsored
	blogPost.SponsorProbability = probability
	blogPost.SponsorIndicators = indicators
	blogPost.Error = "" // 협찬이 확인된 경우 에러 필드 초기화
}

// 이미지 Url에서 협찬 도메인을 확인하는 함수
func CheckSponsorDomain(url string, domains []string) (bool, string) {
	if url == "" {
		return false, ""
	}

	for _, domain := range domains {
		if strings.Contains(url, domain) {
			return true, domain
		}
	}

	return false, ""
}
