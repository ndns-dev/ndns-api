package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sh5080/ndns-go/pkg/configs"
	"github.com/sh5080/ndns-go/pkg/types/structures"
)

// NewCrawlerService는 새 크롤러 서비스를 생성합니다
func NewCrawlerService() CrawlerService {
	return &Service{
		config: configs.GetConfig(),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
func (s *Service) CrawlBlogPost(url string) (*structures.CrawlResult, error) {
	// HTTP 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	// 사용자 에이전트 설정
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// HTTP 요청 실행
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 상태 코드 오류: %d", resp.StatusCode)
	}

	// HTML 문서 로드
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	// 결과 초기화
	result := &structures.CrawlResult{}

	// 네이버 블로그 제목 추출
	result.Title = strings.TrimSpace(doc.Find(".se-title-text").Text())
	if result.Title == "" {
		// 모바일 블로그 또는 구 블로그 형식 처리
		result.Title = strings.TrimSpace(doc.Find(".se_title, .pcol1").Text())
	}

	// 첫 번째 문단 또는 인용구 추출
	firstParagraph := ""

	// 새 블로그 형식 (SE 에디터)
	doc.Find(".se-text-paragraph, .se-quotation").Each(func(i int, s *goquery.Selection) {
		if i == 0 && firstParagraph == "" {
			// 첫번째 문단 추출
			text := strings.TrimSpace(s.Text())
			if len(text) > 0 {
				firstParagraph = text
			}
		}
	})

	// 구 블로그 형식
	if firstParagraph == "" {
		doc.Find(".post-view p, blockquote").Each(func(i int, s *goquery.Selection) {
			if i == 0 && firstParagraph == "" {
				text := strings.TrimSpace(s.Text())
				if len(text) > 0 {
					firstParagraph = text
				}
			}
		})
	}

	result.FirstParagraph = firstParagraph

	// 모든 이미지 URL 추출
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			// 올바른 URL 형식인지 확인
			if strings.HasPrefix(src, "http") {
				result.ImageURL = src
			} else if strings.HasPrefix(src, "//") {
				result.ImageURL = "https:" + src
			}
		}
	})

	// 스티커 이미지 URL 추출 (네이버 블로그 스티커)
	doc.Find(".se-sticker-image img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			// 올바른 URL 형식인지 확인
			if strings.HasPrefix(src, "http") {
				result.StickerURL = src
			} else if strings.HasPrefix(src, "//") {
				result.StickerURL = "https:" + src
			}
		}
	})

	return result, nil
}
