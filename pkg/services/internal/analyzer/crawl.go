package analyzer

import (
	"sync"

	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// Crawl은 크롤링한 값을 분석합니다
func Crawl(posts []structure.BlogPost, ocrExtractor OCRFunc, ocrCache OCRCacheFunc) []structure.BlogPost {
	// 결과 복사 (참조 방지)
	results := make([]structure.BlogPost, len(posts))
	copy(results, posts)

	// 이미 확실한 스폰서로 판단된 포스트가 있는지 확인
	hasDefiniteSponsor := false
	for _, post := range results {
		if post.IsSponsored && post.SponsorProbability >= structure.Accuracy.Exact {
			hasDefiniteSponsor = true
			break
		}
	}

	// 확실한 스폰서가 없는 경우에만 추가 크롤링 수행
	if !hasDefiniteSponsor {
		// 동시성 제어를 위한 WaitGroup과 뮤텍스
		var wg sync.WaitGroup
		// var mu sync.Mutex

		for i, post := range results {
			// 이미 높은 확률의 스폰서로 판단된 경우 크롤링 건너뛰기
			if post.IsSponsored && post.SponsorProbability >= structure.Accuracy.Possible {
				continue
			}

			wg.Add(1)

			// 고루틴으로 크롤링 및 분석 (병렬 처리)
			go func(index int, blogPost structure.BlogPost) {
				defer wg.Done()

			}(i, post)
		}

		// 모든 고루틴이 완료될 때까지 대기
		wg.Wait()
	}

	return results
}

// 필요한 함수 타입 정의
type CrawlerFunc func(url string) (*structure.CrawlResult, error)
type OCRFunc func(imageURL string) (string, error)
type OCRCacheFunc func(imageURL string) (string, bool)
