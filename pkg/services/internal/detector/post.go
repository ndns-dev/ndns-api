package detector

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/analyzer"
	"github.com/sh5080/ndns-go/pkg/services/internal/common"
	"github.com/sh5080/ndns-go/pkg/services/internal/crawler"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// PostImpl는 포스트 감지 서비스 구현체입니다
type PostImpl struct {
	_interface.Service
	ocrService _interface.OcrService
}

// NewPostService는 새 포스트 감지 서비스를 생성합니다
func NewPostService(ocrService _interface.OcrService) _interface.PostService {
	return &PostImpl{
		Service: _interface.Service{
			Client: &http.Client{
				Timeout: time.Second * 30,
			},
			Config: configs.GetConfig(),
		},
		ocrService: ocrService,
	}
}

// DetectPosts는 여러 포스트에서 동시에 협찬 관련 텍스트를 탐지합니다
func (s *PostImpl) DetectPosts(posts []structure.NaverSearchItem) ([]structure.AnalyzedResponse, error) {
	results := make([]structure.AnalyzedResponse, len(posts))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, post := range posts {
		wg.Add(1)
		go func(index int, item structure.NaverSearchItem) {
			defer wg.Done()

			utils.DebugLog("포스트 날짜: %v\n", item.PostDate)
			is2025OrLater := utils.IsAfter2025(item.PostDate)
			blogPost := analyzer.CreateAnalyzedResponse(item)

			// 1. Description 텍스트 탐지 수행
			isSponsored, probability, indicators := common.DetectSponsor(item.Description, structure.SponsorTypeDescription)

			if isSponsored {
				analyzer.UpdateAnalyzedResponseWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
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
					isSponsored, probability, indicators := common.DetectSponsor(crawlResult.FirstParagraph, structure.SponsorTypeParagraph)
					if isSponsored {
						analyzer.UpdateAnalyzedResponseWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 4. 마지막 문단 분석 (2025년 이전 포스트만)
				if !blogPost.IsSponsored && !is2025OrLater && crawlResult.LastParagraph != "" && crawlResult.LastParagraph != crawlResult.FirstParagraph {
					isSponsored, probability, indicators := common.DetectSponsor(crawlResult.LastParagraph, structure.SponsorTypeParagraph)
					if isSponsored {
						analyzer.UpdateAnalyzedResponseWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					}
				}

				// 5. 이미지 URL에서 협찬 도메인 패턴 확인
				if !blogPost.IsSponsored {
					isSponsored, probability, indicators := common.CheckSponsorImagesInCrawlResult(crawlResult)
					if isSponsored {
						analyzer.UpdateAnalyzedResponseWithSponsorInfo(&blogPost, isSponsored, probability, indicators)
					} else {
						// 6. 이미지 URL에서 협찬이 발견되지 않은 경우 Ocr 요청 전송 시작
						jobId := uuid.New().String()
						// Ocr 요청 상태 표시
						pendingIndicator := CreatePendingIndicator(jobId)
						blogPost.SponsorIndicators = append(blogPost.SponsorIndicators, pendingIndicator)

						// Ocr 요청 상태 초기화
						state := model.OcrQueueState{
							JobId:           jobId,
							CrawlResult:     crawlResult,
							CurrentPosition: model.OcrPositionFirstImage,
							Is2025OrLater:   is2025OrLater,
						}

						// 첫 번째 Ocr 요청
						err := s.ocrService.RequestNextOcr(state)
						if err != nil {
							utils.DebugLog("Ocr 요청 실패: %v\n", err)
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
