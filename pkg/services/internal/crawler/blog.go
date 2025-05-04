package crawler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sh5080/ndns-go/pkg/configs"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// CrawlerService는 블로그 콘텐츠를 크롤링하는 인터페이스입니다
type CrawlerService interface {
	// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
	CrawlBlogPost(url string) (*structure.CrawlResult, error)
}

// CrawlerImpl는 블로그 크롤러 구현체입니다
type CrawlerImpl struct {
	client *http.Client
	config *configs.EnvConfig
}

// NewCrawlerService는 새 크롤러 서비스를 생성합니다
func NewCrawlerService() CrawlerService {
	return &CrawlerImpl{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		config: configs.GetConfig(),
	}
}

// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
func (c *CrawlerImpl) CrawlBlogPost(url string) (*structure.CrawlResult, error) {
	if url == "" {
		return nil, fmt.Errorf("URL이 비어 있습니다")
	}

	// URL 정규화
	url = normalizeURL(url)

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 요청 헤더 추가 (브라우저 에뮬레이션)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

	// 요청 실행
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 오류 (%d)", resp.StatusCode)
	}

	// HTML 파싱
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	// 결과 초기화
	result := &structure.CrawlResult{
		URL: url,
	}

	// 블로그 타입 감지 및 파싱
	if strings.Contains(url, "blog.naver.com") {
		c.parseNaverBlog(doc, result)
	} else if strings.Contains(url, "tistory.com") {
		c.parseTistoryBlog(doc, result)
	} else {
		c.parseGenericBlog(doc, result)
	}

	return result, nil
}

// parseNaverBlog는 네이버 블로그 HTML을 파싱합니다
func (c *CrawlerImpl) parseNaverBlog(doc *goquery.Document, result *structure.CrawlResult) {
	// 제목 추출
	result.Title = strings.TrimSpace(doc.Find("div.se-module-text h3, .se_title, .htitle").First().Text())

	// 스티커 이미지 추출 (네이버 블로그 스티커)
	doc.Find("img.se-sticker-image, img.se-image-resource").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.StickerURL == "" {
			if src, exists := s.Attr("src"); exists {
				result.StickerURL = src
			}
		}
	})

	// 일반 이미지 추출
	doc.Find("img.se-image-resource, .se-image-resource").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.ImageURL == "" {
			if src, exists := s.Attr("src"); exists {
				result.ImageURL = src
			}
		}
	})

	// 첫 문단 추출
	doc.Find("div.se-main-container p, div.se_component_wrap p").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.FirstParagraph == "" {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				result.FirstParagraph = text
			}
		}
	})
}

// parseTistoryBlog는 티스토리 블로그 HTML을 파싱합니다
func (c *CrawlerImpl) parseTistoryBlog(doc *goquery.Document, result *structure.CrawlResult) {
	// 제목 추출
	result.Title = strings.TrimSpace(doc.Find("h1.title, .post-title, .entry-title").First().Text())

	// 이미지 추출
	doc.Find("article img, .article img, .entry-content img").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.ImageURL == "" {
			if src, exists := s.Attr("src"); exists {
				result.ImageURL = src
			}
		}
	})

	// 첫 문단 추출
	doc.Find("article p, .article p, .entry-content p").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.FirstParagraph == "" {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				// HTML 태그 제거
				result.FirstParagraph = utils.RemoveHTMLTags(text)
			}
		}
	})
}

// parseGenericBlog는 일반 블로그 HTML을 파싱합니다
func (c *CrawlerImpl) parseGenericBlog(doc *goquery.Document, result *structure.CrawlResult) {
	// 제목 추출
	result.Title = strings.TrimSpace(doc.Find("h1, h2, .title, .post-title, .entry-title").First().Text())

	// 이미지 추출
	doc.Find("article img, .article img, .content img, .post img, .entry img").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.ImageURL == "" {
			if src, exists := s.Attr("src"); exists {
				result.ImageURL = src
			}
		}
	})

	// 첫 문단 추출
	doc.Find("article p, .article p, .content p, .post p, .entry p").Each(func(i int, s *goquery.Selection) {
		if i == 0 && result.FirstParagraph == "" {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				// HTML 태그 제거
				result.FirstParagraph = utils.RemoveHTMLTags(text)
			}
		}
	})
}

// normalizeURL은 URL을 정규화합니다
func normalizeURL(url string) string {
	// HTTP/HTTPS 접두사 추가
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return url
}
