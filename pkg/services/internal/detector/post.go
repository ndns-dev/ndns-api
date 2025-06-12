package detector

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/analyzer"
	"github.com/sh5080/ndns-go/pkg/services/internal/crawler"
	constant "github.com/sh5080/ndns-go/pkg/types"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// OCR 처리 공통 함수
func processOCR(url string, ocrExtractor _interface.OCRFunc, sourceType structure.SponsorType) (bool, float64, []structure.SponsorIndicator, string) {
	// URL이 비어있으면 처리 건너뜀
	if url == "" {
		return false, 0, nil, ""
	}

	ocrText, err := ocrExtractor(url)

	if err != nil {
		errMsg := fmt.Sprintf("OCR 처리 오류: %s", err.Error())
		utils.DebugLog("OCR 오류: %s\n", err.Error())
		return false, 0, nil, errMsg
	}

	if strings.Contains(ocrText, "context deadline exceeded") || strings.Contains(ocrText, "Get \"") {
		return false, 0, nil, ocrText
	}

	trimmedText := strings.TrimSpace(ocrText)
	textLength := len(trimmedText)

	// 한글 단어(2글자 이상) 포함 확인
	hangulRegex := regexp.MustCompile(`[가-힣]{2,}`)

	// 스티커 타입에 대한 특별 처리
	if sourceType == structure.SponsorTypeSticker {
		// 1. 먼저 한글 텍스트가 있는지 확인
		if hangulRegex.MatchString(trimmedText) {
			isSponsored, probability, indicators := DetectSponsor(trimmedText, sourceType)
			return isSponsored, probability, indicators, ""
		}

		// 3. 위 조건에 모두 해당하지 않고 텍스트가 너무 짧은 경우
		if textLength < 10 {
			utils.DebugLog("스티커 OCR 텍스트가 너무 짧고 의미 없음 (%d자): %s\n", textLength, trimmedText)
			return false, 0, nil, "OCR_TEXT_TOO_SHORT"
		}
	}

	// 일반적인 경우 처리
	isSponsored, probability, indicators := DetectSponsor(ocrText, sourceType)
	return isSponsored, probability, indicators, ""
}

// DetectTextInPosts는 여러 포스트에서 동시에 협찬 관련 텍스트를 탐지합니다
func DetectTextInPosts(posts []structure.NaverSearchItem, ocrExtractor _interface.OCRFunc) []structure.BlogPost {
	// 결과를 저장할 슬라이스 초기화
	results := make([]structure.BlogPost, len(posts))

	// 모든 결과 항목 미리 초기화
	for i, post := range posts {
		results[i] = analyzer.CreateBlogPost(post)
	}

	// 동시성 제어를 위한 WaitGroup
	var wg sync.WaitGroup

	// 동시성 제어를 위한 뮤텍스
	var mu sync.Mutex

	// 각 포스트에 대해 병렬로 처리
	for i, post := range posts {
		wg.Add(1)

		// 고루틴으로 포스트 분석
		go func(index int, item structure.NaverSearchItem) {
			defer wg.Done()

			utils.DebugLog("포스트 날짜: %v\n", item.PostDate)
			// 2025년 이후 포스트인지 확인
			is2025OrLater := utils.IsAfter2025(item.PostDate)

			// 블로그 포스트 초기화 (analyzer 패키지 사용)
			blogPost := analyzer.CreateBlogPost(item)

			// 1. Description 텍스트 탐지 수행
			isSponsored, probability, indicators := DetectSponsor(item.Description, structure.SponsorTypeDescription)

			if isSponsored {
				// 공통 함수 사용하여 스폰서 정보 업데이트
				analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
			} else {
				// 2. Description에서 스폰서 탐지 실패시 본문 크롤링
				crawlResult, err := crawler.CrawlBlogPost(item.Link, is2025OrLater)
				if err != nil {
					fmt.Printf("[%d] 크롤링 실패: %v\n", index, err)
					// 크롤링 실패 시 에러 메시지 저장하고 결과 반환
					blogPost.Error = fmt.Sprintf("크롤링 실패: %v", err)
					mu.Lock()
					results[index] = blogPost
					mu.Unlock()
					return
				}

				// crawlResult가 nil인 경우 처리
				if crawlResult == nil {
					blogPost.Error = "크롤링 결과가 없습니다"
					mu.Lock()
					results[index] = blogPost
					mu.Unlock()
					return
				}

				// 본문 분석 순서:
				// 1. 첫 이미지/스티커 URL 도메인 확인
				// 2. 도메인이 협찬이 아니면 첫 이미지/스티커 OCR 분석
				// 3. 첫 문단 분석
				// 4. 2025년 이전 포스트만: 마지막 문단/스티커/이미지 분석

				// 1. 첫 번째 이미지 URL과 스티커 URL 도메인 확인
				foundSponsorDomain := false
				var foundURL, domain string
				sponsorType := structure.SponsorTypeImage

				// 1-1. 첫 번째 이미지 URL 확인
				utils.DebugLog("1-1. 첫 번째 이미지 URL 확인\n")
				if foundDomain, matchedDomain := analyzer.CheckSponsorDomain(crawlResult.FirstImageURL, constant.SPONSOR_DOMAINS); foundDomain {
					foundSponsorDomain = true
					foundURL = crawlResult.FirstImageURL
					sponsorType = structure.SponsorTypeImage
					domain = matchedDomain
				}

				utils.DebugLog("1-2. 첫 번째 스티커 URL 확인\n")
				// 1-2. 첫 번째 스티커 URL 확인 (첫 번째 이미지 URL에서 발견되지 않은 경우)
				if !foundSponsorDomain && crawlResult.FirstStickerURL != "" {
					if foundDomain, matchedDomain := analyzer.CheckSponsorDomain(crawlResult.FirstStickerURL, constant.SPONSOR_DOMAINS); foundDomain {
						foundSponsorDomain = true
						foundURL = crawlResult.FirstStickerURL
						sponsorType = structure.SponsorTypeSticker
						domain = matchedDomain
					}
				}

				// 1-3. 협찬 도메인이 발견된 경우
				if foundSponsorDomain {
					// 협찬 도메인이 발견되었으므로 바로 협찬으로 판단
					blogPost = analyzer.CreateSponsoredBlogPost(
						item,
						structure.Accuracy.Absolute,
						foundURL,
						structure.IndicatorTypeKeyword,
						structure.PatternTypeNormal,
						sponsorType,
						domain,
					)

					// 중요: 도메인으로 협찬이 확인된 경우 에러 필드를 명시적으로 비웁니다
					blogPost.Error = ""

					// 결과 저장
					mu.Lock()
					results[index] = blogPost
					mu.Unlock()
					return
				}

				// 2. 도메인에서 협찬이 발견되지 않은 경우, 이미지/스티커 OCR 분석
				// 2-1. 첫 번째 이미지 OCR 처리
				if crawlResult.FirstImageURL != "" && !blogPost.IsSponsored && blogPost.Error == "" {
					utils.DebugLog("2-1. 첫 번째 이미지 OCR 처리\n")
					isSponsored, probability, indicators, errMsg := processOCR(crawlResult.FirstImageURL, ocrExtractor, structure.SponsorTypeImage)

					if errMsg != "" {
						// 오류 메시지 저장
						analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, false, 0, nil, errMsg)
					} else if isSponsored {
						// 협찬 정보 업데이트
						analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 2-2. 첫 번째 스티커 OCR 처리 (첫 번째 이미지에서 스폰서가 발견되지 않은 경우)
				if crawlResult.FirstStickerURL != "" && !blogPost.IsSponsored && blogPost.Error == "" {
					utils.DebugLog("2-2. 첫 번째 스티커 OCR 처리\n")
					isSponsored, probability, indicators, errMsg := processOCR(crawlResult.FirstStickerURL, ocrExtractor, structure.SponsorTypeSticker)

					// 첫 번째 스티커 OCR 결과가 너무 짧은 경우, 두 번째 스티커 시도
					if errMsg == "OCR_TEXT_TOO_SHORT" && crawlResult.SecondStickerURL != "" && crawlResult.SecondStickerURL != crawlResult.FirstStickerURL {
						utils.DebugLog("첫 번째 스티커 OCR 텍스트가 너무 짧아 두 번째 스티커 처리\n")
						isSponsored, probability, indicators, errMsg = processOCR(crawlResult.SecondStickerURL, ocrExtractor, structure.SponsorTypeSticker)
					}

					if errMsg != "" && errMsg != "OCR_TEXT_TOO_SHORT" {
						// 오류 메시지 저장
						analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, false, 0, nil, errMsg)
					} else if isSponsored {
						// 협찬 정보 업데이트
						analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 3-1. 첫 번째 문단 분석 (이미지/스티커 OCR에서 스폰서가 발견되지 않은 경우)
				if !blogPost.IsSponsored && blogPost.Error == "" {
					utils.DebugLog("3-1. 첫 번째 문단 분석\n")
					isSponsored, probability, indicators := DetectSponsor(crawlResult.FirstParagraph, structure.SponsorTypeParagraph)

					if isSponsored {
						//협찬 정보 업데이트
						analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					} else if !is2025OrLater { // 2025년 이전 포스트만 추가 분석 수행
						// 3-2. 마지막 문단 분석 (첫 문단과 다른 경우만)
						utils.DebugLog("3-2. 마지막 문단 분석 (2025년 이전 포스트만)\n")
						if crawlResult.LastParagraph != "" && crawlResult.LastParagraph != crawlResult.FirstParagraph {
							isSponsored, probability, indicators = DetectSponsor(crawlResult.LastParagraph, structure.SponsorTypeParagraph)
							if isSponsored {
								//협찬 정보 업데이트
								analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
							}
						}

						// 4. 마지막 스티커 이미지 OCR 처리 (협찬이 발견되지 않은 경우)
						utils.DebugLog("4-1. 마지막 스티커 이미지 OCR 처리 (2025년 이전 포스트만)\n")
						if !blogPost.IsSponsored && blogPost.Error == "" && crawlResult.LastStickerURL != "" && crawlResult.LastStickerURL != crawlResult.FirstStickerURL {
							// 마지막 스티커 URL이 협찬 도메인인지 먼저 확인
							if foundDomain, matchedDomain := analyzer.CheckSponsorDomain(crawlResult.LastStickerURL, constant.SPONSOR_DOMAINS); foundDomain {
								// 협찬 도메인이 발견된 경우 바로 협찬으로 판단
								analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, true, structure.Accuracy.Absolute, []structure.SponsorIndicator{
									analyzer.CreateSponsorIndicator(
										structure.IndicatorTypeKeyword,
										structure.PatternTypeNormal,
										crawlResult.LastStickerURL,
										structure.Accuracy.Absolute,
										structure.SponsorTypeSticker,
										matchedDomain,
									),
								})
							} else {
								// 협찬 도메인이 아닌 경우 OCR 처리 진행
								isSponsored, probability, indicators, errMsg := processOCR(crawlResult.LastStickerURL, ocrExtractor, structure.SponsorTypeSticker)

								if errMsg != "" {
									// 오류 메시지 저장
									analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, false, 0, nil, errMsg)
								} else if isSponsored {
									// 협찬 정보 업데이트
									analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
								}
							}
						}

						// 4-2. 마지막 이미지 OCR 처리 (협찬이 발견되지 않은 경우)
						utils.DebugLog("4-2. 마지막 이미지 OCR 처리 (2025년 이전 포스트만)\n")
						// 마지막 이미지 URL이 비어있지 않고 첫 번째 이미지 URL과 다르면 협찬 탐지 진행
						if !blogPost.IsSponsored && crawlResult.LastImageURL != "" && crawlResult.LastImageURL != crawlResult.FirstImageURL {
							// 마지막 이미지 URL이 협찬 도메인인지 먼저 확인
							if foundDomain, matchedDomain := analyzer.CheckSponsorDomain(crawlResult.LastImageURL, constant.SPONSOR_DOMAINS); foundDomain {
								// 협찬 도메인이 발견된 경우 바로 협찬으로 판단

								analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, true, structure.Accuracy.Absolute, []structure.SponsorIndicator{
									analyzer.CreateSponsorIndicator(
										structure.IndicatorTypeKeyword,
										structure.PatternTypeNormal,
										crawlResult.LastImageURL,
										structure.Accuracy.Absolute,
										structure.SponsorTypeImage,
										matchedDomain,
									),
								})
							} else {
								// 협찬 도메인이 아닌 경우 OCR 처리 진행
								isSponsored, probability, indicators, errMsg := processOCR(crawlResult.LastImageURL, ocrExtractor, structure.SponsorTypeImage)

								if errMsg != "" {
									// 오류 메시지 저장
									analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, false, 0, nil, errMsg)
								} else if isSponsored {
									// 협찬 정보 업데이트
									analyzer.UpdateBlogPostWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
								}
							}
						}
					} else {
						utils.DebugLog("2025년 이후 포스트이므로 마지막 문단/스티커/이미지 분석 건너뜀\n")
					}
				}
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
