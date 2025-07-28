package analyzer

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sh5080/ndns-go/pkg/services/internal/crawler"
	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	utils "github.com/sh5080/ndns-go/pkg/utils"
)

// UpdateAnalyzedResponse는 협찬 감지 결과를 블로그 포스트에 업데이트합니다
func updateAnalyzedResponse(
	blogPost *responseDto.AnalyzedResponse,
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

// AnalyzePosts는 여러 포스트에서 동시에 협찬 관련 텍스트를 분석합니다
func (s *AnalyzerService) AnalyzePosts(posts []structure.NaverSearchItem, reqId string) ([]responseDto.AnalyzedResponse, error) {
	results := make([]responseDto.AnalyzedResponse, len(posts))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, post := range posts {
		wg.Add(1)
		go func(index int, item structure.NaverSearchItem) {
			defer wg.Done()
			is2025OrLater := utils.IsAfter2025(item.PostDate)
			blogPost := responseDto.AnalyzedResponse{
				NaverSearchItem:    item,
				IsSponsored:        false,
				SponsorProbability: 0,
				SponsorIndicators:  []structure.SponsorIndicator{},
			}

			// 1. Description 텍스트 탐지 수행
			isSponsored, probability, indicators := detector.DetectSponsor(item.Description, structure.SponsorTypeDescription)

			if isSponsored {
				updateAnalyzedResponse(&blogPost, isSponsored, probability, indicators)
			} else {
				// 2. Description에서 스폰서 탐지 실패시 본문 크롤링
				crawlResult, err := crawler.CrawlAnalyzedResponse(item.Link, is2025OrLater)
				if err != nil {
					fmt.Printf("[%d] 크롤링 실패: %v\n", index, err)
					blogPost.Error = fmt.Sprintf("크롤링 실패: %v", err)
					mu.Lock()
					results[index] = blogPost
					mu.Unlock()
					return
				}

				if crawlResult == nil {
					blogPost.Error = "크롤링 결과가 없습니다"
					mu.Lock()
					results[index] = blogPost
					mu.Unlock()
					return
				}

				// 3. 첫 번째 문단 분석
				if !blogPost.IsSponsored {
					isSponsored, probability, indicators := detector.DetectSponsor(crawlResult.FirstParagraph, structure.SponsorTypeParagraph)
					if isSponsored {
						updateAnalyzedResponse(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 4. 마지막 문단 분석 (2025년 이전 포스트만)
				if !blogPost.IsSponsored && !is2025OrLater && crawlResult.LastParagraph != "" && crawlResult.LastParagraph != crawlResult.FirstParagraph {
					isSponsored, probability, indicators := detector.DetectSponsor(crawlResult.LastParagraph, structure.SponsorTypeParagraph)
					if isSponsored {
						updateAnalyzedResponse(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 5. 이미지 URL에서 협찬 도메인 패턴 확인
				if !blogPost.IsSponsored {
					isSponsored, probability, indicators := detector.CheckSponsorImagesInCrawlResult(crawlResult)
					if isSponsored {
						updateAnalyzedResponse(&blogPost, isSponsored, probability, indicators)
					} else {
						// 6. 이미지 URL에서 협찬이 발견되지 않은 경우 Ocr 요청 전송 시작
						jobId := uuid.New().String()
						// Ocr 요청 상태 표시
						pendingIndicator := detector.CreatePendingIndicator(jobId)
						blogPost.SponsorIndicators = append(blogPost.SponsorIndicators, pendingIndicator)

						// Ocr 요청 상태 초기화
						state := model.OcrQueueState{
							JobId:           jobId,
							ReqId:           reqId,
							CrawlResult:     crawlResult,
							CurrentPosition: model.OcrPositionStart,
							Is2025OrLater:   is2025OrLater,
						}

						utils.WebhookLog("[ndns-go]OCR 요청 시작: %+v\n", state)
						// 첫 번째 OCR 요청
						err := s.detectorService.RequestNextOcr(state)
						if err != nil {
							utils.DebugLog("[ndns-go]OCR 요청 실패: %v\n", err)
						}
					}
				}
			}
			mu.Lock()
			results[index] = blogPost
			mu.Unlock()
		}(i, post)
	}

	wg.Wait()
	return results, nil
}

// GetExistingPosts는 기존 분석결과를 조회합니다
func (s *AnalyzerService) GetExistingPosts(posts []structure.NaverSearchItem) ([]responseDto.AnalyzedResponse, error) {
	utils.DebugLog("GetExistingPosts 시작: %d개 포스트 조회\n", len(posts))

	// 모든 링크를 수집
	links := make([]string, len(posts))
	for i, post := range posts {
		links[i] = post.Link
		utils.DebugLog("조회할 링크[%d]: %s\n", i, post.Link)
	}

	// BatchGetItem으로 한 번에 조회
	analyzedResults, err := s.analyzedResultRepository.GetAnalyzedResults(links)
	if err != nil {
		utils.DebugLog("GetExistingPosts 배치 조회 실패: %v\n", err)
		return nil, err
	}

	utils.DebugLog("DB에서 조회된 결과: %d개\n", len(analyzedResults))
	for link, result := range analyzedResults {
		utils.DebugLog("DB 결과 - 링크: %s, IsSponsored: %v\n", link, result.IsSponsored)
	}

	// 실제 존재하는 분석 결과만 수집
	results := make([]responseDto.AnalyzedResponse, 0)
	for _, post := range posts {
		if analyzedResult, exists := analyzedResults[post.Link]; exists && analyzedResult != nil {
			utils.DebugLog("기존 결과 발견: %s\n", post.Link)
			results = append(results, responseDto.AnalyzedResponse{
				NaverSearchItem:    post,
				IsSponsored:        analyzedResult.IsSponsored,
				SponsorProbability: analyzedResult.SponsorProbability,
				SponsorIndicators:  analyzedResult.SponsorIndicators,
			})
		} else {
			utils.DebugLog("기존 결과 없음: %s\n", post.Link)
		}
	}

	utils.DebugLog("GetExistingPosts 완료: %d개 결과 반환\n", len(results))
	return results, nil
}
