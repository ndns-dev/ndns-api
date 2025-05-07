package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	constants "github.com/sh5080/ndns-go/pkg/types"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
func CrawlBlogPost(url string) (*structure.CrawlResult, error) {
	if url == "" {
		return nil, fmt.Errorf("URL이 비어 있습니다")
	}

	// URL 정규화
	url = normalizeURL(url)
	fmt.Printf("크롤링 시작: %s\n", url)

	// 결과 초기화
	result := &structure.CrawlResult{
		URL: url,
	}

	// 네이버 블로그인 경우 특별 처리
	if strings.Contains(url, "blog.naver.com") {
		// 먼저 프레임셋 페이지 가져오기
		framesetDoc, err := fetchHTML(url)
		if err != nil {
			return nil, fmt.Errorf("프레임셋 페이지 가져오기 실패: %v", err)
		}

		// iframe 태그에서 실제 콘텐츠 URL 추출
		iframeURL := extractNaverIframeURL(framesetDoc, url)
		if iframeURL != "" {
			contentDoc, err := fetchHTML(iframeURL)
			if err != nil {
				return nil, fmt.Errorf("iframe 내부 콘텐츠 가져오기 실패: %v", err)
			}
			parseNaverBlog(contentDoc, result)

		}
	} else if strings.Contains(url, "tistory.com") {
		// 티스토리 블로그 크롤링
		doc, err := fetchHTML(url)
		if err != nil {
			return nil, err
		}
		parseTistoryBlog(doc, result)
	} else {
		// 기타 블로그 크롤링
		doc, err := fetchHTML(url)
		if err != nil {
			return nil, err
		}
		parseGenericBlog(doc, result)
	}

	return result, nil
}

// fetchHTML은 URL에서 HTML을 가져와 goquery.Document로 반환합니다
func fetchHTML(url string) (*goquery.Document, error) {
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
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 오류 (%d)", resp.StatusCode)
	}

	// HTML 내용 읽기
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 본문 읽기 실패: %v", err)
	}
	bodyHTML := string(bodyBytes)

	if len(bodyHTML) == 0 {
		return nil, fmt.Errorf("응답 HTML이 비어있습니다")
	}

	// HTML 파싱을 위해 Reader 생성
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyHTML))
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	return doc, nil
}

// extractNaverIframeURL은 네이버 블로그 프레임셋에서 실제 콘텐츠 iframe URL을 추출합니다
func extractNaverIframeURL(doc *goquery.Document, originalURL string) string {
	// mainFrame의 src 속성 확인
	iframeSrc := ""
	doc.Find("#mainFrame, iframe[name='mainFrame']").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			iframeSrc = src
			return
		}
	})

	// src가 상대 경로인 경우 절대 경로로 변환
	if iframeSrc != "" && !strings.HasPrefix(iframeSrc, "http") {
		if strings.HasPrefix(iframeSrc, "/") {
			// 도메인만 추출
			urlParts := strings.Split(originalURL, "/")
			domain := ""
			if len(urlParts) >= 3 {
				domain = urlParts[0] + "//" + urlParts[2]
			} else {
				domain = "https://blog.naver.com"
			}
			iframeSrc = domain + iframeSrc
		} else {
			// 기본 도메인 사용
			iframeSrc = "https://blog.naver.com/" + iframeSrc
		}
	}

	return iframeSrc
}

// parseNaverBlog는 네이버 블로그 HTML을 파싱합니다
func parseNaverBlog(doc *goquery.Document, result *structure.CrawlResult) {
	// 스티커 이미지 추출
	extractFirstSticker(doc, result)
	// 일반 이미지 추출
	extractFirstImage(doc, result)
	// 첫 문단 추출
	extractFirstParagraph(doc, result)
}

// extractFirstSticker는 첫 번째 스티커를 추출합니다
func extractFirstSticker(doc *goquery.Document, result *structure.CrawlResult) {
	// 1. 스티커 클래스로 찾기
	for _, stickerClass := range constants.STICKER_CLASSES {
		doc.Find("[class*='" + stickerClass + "']").Each(func(i int, elem *goquery.Selection) {
			if result.StickerURL != "" {
				return
			}

			// 이미지 태그 찾기
			img := elem.Find("img")
			if img.Length() > 0 && img.AttrOr("src", "") != "" {
				imgURL := img.AttrOr("src", "")
				// 스티커 도메인 확인
				for _, domain := range constants.STICKER_DOMAINS {
					if strings.Contains(imgURL, domain) {
						result.StickerURL = imgURL
						return
					}
				}
			}

			// 배경 이미지 스타일 확인
			style := elem.AttrOr("style", "")
			if strings.Contains(style, "background-image") {
				urlRegex := regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)
				matches := urlRegex.FindStringSubmatch(style)
				if len(matches) > 1 {
					imgURL := matches[1]
					// 스티커 도메인 확인
					for _, domain := range constants.STICKER_DOMAINS {
						if strings.Contains(imgURL, domain) {
							result.StickerURL = imgURL
							return
						}
					}
				}
			}
		})

		if result.StickerURL != "" {
			break
		}
	}

	// 2. data-linkdata 속성으로 찾기 (네이버 블로그 특유의 구조)
	if result.StickerURL == "" {
		doc.Find("[data-linkdata]").Each(func(i int, elem *goquery.Selection) {
			if result.StickerURL != "" {
				return
			}

			linkData := elem.AttrOr("data-linkdata", "")
			if linkData != "" {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(linkData), &data)
				if err == nil && data["src"] != nil {
					imgURL := data["src"].(string)
					// 스티커 도메인 확인
					for _, domain := range constants.STICKER_DOMAINS {
						if strings.Contains(imgURL, domain) {
							result.StickerURL = imgURL
							return
						}
					}
				}
			}
		})
	}

	// 3. 이미지 태그 확인
	if result.StickerURL == "" {
		doc.Find("img").Each(func(i int, img *goquery.Selection) {
			if result.StickerURL != "" {
				return
			}

			imgURL := img.AttrOr("src", "")
			// 스티커 도메인 확인
			for _, domain := range constants.STICKER_DOMAINS {
				if strings.Contains(imgURL, domain) {
					result.StickerURL = imgURL
					return
				}
			}
		})
	}
}

// extractFirstImage는 첫 번째 이미지를 본문 영역에서만 추출합니다
func extractFirstImage(doc *goquery.Document, result *structure.CrawlResult) {

	// 본문 영역 찾기
	var contentArea *goquery.Selection
	for _, selector := range constants.CONTENT_SELECTORS {
		contentArea = doc.Find(selector).First()
		if contentArea.Length() > 0 {
			break
		}
	}

	// 본문 영역을 찾지 못한 경우 전체 HTML 사용
	if contentArea == nil || contentArea.Length() == 0 {
		contentArea = doc.Selection
	}

	// 1. 스마트에디터 이미지 리소스 확인 (스티커가 아닌 이미지만)
	contentArea.Find(".se-image-resource").Each(func(i int, img *goquery.Selection) {
		if result.ImageURL != "" {
			return
		}

		// 상위 요소가 스티커 모듈이 아닌지 확인
		if img.ParentsFiltered(".se-module-sticker").Length() > 0 {
			return // 스티커 모듈 내부의 이미지는 건너뜀
		}

		imgURL := img.AttrOr("src", "")
		if imgURL != "" && !containsStickerDomain(imgURL) {
			result.ImageURL = imgURL
		}
	})

	// 2. se-component 이미지 확인 (스티커가 아닌 이미지만)
	if result.ImageURL == "" {
		contentArea.Find(".se-component.se-image").Each(func(i int, component *goquery.Selection) {
			if result.ImageURL != "" {
				return
			}

			// 이미지 모듈 찾기
			img := component.Find(".se-module-image .se-image-resource").First()
			if img.Length() > 0 {
				imgURL := img.AttrOr("src", "")
				if imgURL != "" && !containsStickerDomain(imgURL) {
					result.ImageURL = imgURL
					return
				}
			}

			// 링크 데이터 확인
			link := component.Find(".se-module-image-link").First()
			if link.Length() > 0 {
				linkData := link.AttrOr("data-linkdata", "")
				if linkData != "" {
					var data map[string]interface{}
					err := json.Unmarshal([]byte(linkData), &data)
					if err == nil && data["src"] != nil {
						imgURL := data["src"].(string)
						if !containsStickerDomain(imgURL) {
							result.ImageURL = imgURL
						}
					}
				}
			}
		})
	}

	// 3. 일반 이미지 확인 (스티커가 아닌 이미지만)
	if result.ImageURL == "" {
		contentArea.Find("img").Each(func(i int, img *goquery.Selection) {
			if result.ImageURL != "" {
				return
			}

			// 상위 요소가 스티커 관련 요소가 아닌지 확인
			if img.ParentsFiltered("[class*='sticker']").Length() > 0 {
				return // 스티커 관련 요소 내부의 이미지는 건너뜀
			}

			imgURL := img.AttrOr("src", "")
			if imgURL == "" {
				imgURL = img.AttrOr("data-src", "")
			}
			if imgURL == "" {
				imgURL = img.AttrOr("data-lazy-src", "")
			}

			if imgURL != "" && !containsStickerDomain(imgURL) &&
				(strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://")) {
				result.ImageURL = imgURL
			}
		})
	}
	if result.ImageURL != "" && strings.HasSuffix(result.ImageURL, "w80_blur") {
		result.ImageURL = strings.Replace(result.ImageURL, "w80_blur", "w773", 1)
	}
}

// extractFirstParagraph는 첫 번째 문단과 인용구를 추출합니다
func extractFirstParagraph(doc *goquery.Document, result *structure.CrawlResult) {
	// 본문 영역 찾기
	var contentArea *goquery.Selection

	// 각 선택자별로 확인 및 내용 출력
	for _, selector := range constants.CONTENT_SELECTORS {
		selected := doc.Find(selector)
		count := selected.Length()

		if count > 0 {
			contentArea = selected.First()
			break
		}
	}

	// 본문 영역을 찾지 못한 경우 전체 HTML 사용
	if contentArea == nil || contentArea.Length() == 0 {
		fmt.Printf("본문 영역을 찾지 못해 전체 HTML을 사용합니다.\n")
		contentArea = doc.Selection
	}

	// 인용구 확인
	quotationText := ""
	quotationFound := false

	// 인용구 선택자 확인 (네이버 블로그 스마트에디터 패턴에 맞게 개선)
	quotationSelectors := []string{
		".se-quotation-container", // 스마트에디터 2.0 인용구
		"blockquote",              // 일반 인용구
	}

	for _, selector := range quotationSelectors {
		quotes := contentArea.Find(selector)

		if quotes.Length() > 0 {
			quotes.EachWithBreak(func(i int, quote *goquery.Selection) bool {
				if i >= 2 { // 처음 2개까지만
					return false
				}

				// 텍스트 추출 (span 내부까지 확인)
				text := ""

				// 인용구 내부의 span 태그 확인 (색상 등 스타일 적용된 텍스트)
				spans := quote.Find("span")
				if spans.Length() > 0 {
					spans.Each(func(j int, span *goquery.Selection) {
						spanText := strings.TrimSpace(span.Text())
						if spanText != "" && !strings.Contains(text, spanText) {
							if text != "" {
								text += " "
							}
							text += spanText
						}
					})
				}

				// span에서 텍스트를 찾지 못했으면 직접 텍스트 추출
				if text == "" {
					text = strings.TrimSpace(quote.Text())
				}

				// 텍스트 정리 (특수문자 제거)
				text = cleanText(text)

				if text != "" && len(text) > 5 {
					quotationText = text
					quotationFound = true
					return false
				}
				return true
			})
		}

		if quotationFound {
			break
		}
	}

	if !quotationFound {
		fmt.Printf("인용구를 찾지 못했습니다.\n")
	}

	// 일반 문단 확인
	paragraphSelectors := []string{
		".se-text-paragraph", // 스마트에디터 2.0 문단
		".se-module-text p",  // 스마트에디터 모듈 내 문단
		".post_ct p",         // 일반 모바일 블로그 문단
		".sect_dsc p",        // 모바일 본문 문단
		"p",                  // 일반 문단 태그
	}

	firstParagraph := ""
	paragraphFound := false

	for _, selector := range paragraphSelectors {
		paragraphElements := contentArea.Find(selector)
		paragraphCount := paragraphElements.Length()

		if paragraphCount > 0 {
			paragraphElements.EachWithBreak(func(i int, p *goquery.Selection) bool {
				// 텍스트 추출 시 span 내부도 확인
				text := ""
				spans := p.Find("span")
				if spans.Length() > 0 {
					spans.Each(func(j int, span *goquery.Selection) {
						spanText := strings.TrimSpace(span.Text())
						if spanText != "" && !strings.Contains(text, spanText) {
							if text != "" {
								text += " "
							}
							text += spanText
						}
					})
				}

				// span에서 텍스트를 찾지 못했으면 직접 텍스트 추출
				if text == "" {
					text = strings.TrimSpace(p.Text())
				}

				// 텍스트 정리 (특수문자 제거)
				text = cleanText(text)

				if text != "" && len(text) > 5 && text != "" {
					firstParagraph = text
					paragraphFound = true
					return false
				}
				return true
			})
		}

		if paragraphFound {
			fmt.Printf("적합한 문단을 찾았습니다!\n")
			break
		}
	}

	// 문단을 찾지 못한 경우 대체 방법 시도
	if !paragraphFound {
		fmt.Printf("표준 선택자로 문단을 찾지 못했습니다. 다른 방법 시도 중...\n")

		// 1. div 직접 검색
		contentArea.Find("div.se-module-text").Each(func(i int, div *goquery.Selection) {
			if i > 10 || paragraphFound { // 처음 10개만 확인
				return
			}

			// 먼저 span 내부 확인
			var texts []string
			div.Find("span").Each(func(j int, span *goquery.Selection) {
				spanText := strings.TrimSpace(span.Text())
				if spanText != "" && len(spanText) > 5 {
					texts = append(texts, spanText)
				}
			})

			text := strings.Join(texts, " ")

			// span이 없거나 비어있으면 div 자체 텍스트 사용
			if text == "" {
				text = strings.TrimSpace(div.Text())
			}

			// 텍스트 정리
			text = cleanText(text)

			if text != "" && len(text) > 30 && len(text) < 500 {
				fmt.Printf("DIV에서 가능한 문단 발견 (%d 바이트): %s...\n", len(text), text[:min(len(text), 100)])
				if firstParagraph == "" {
					firstParagraph = text
					paragraphFound = true
				}
			}
		})
	}

	// 문단과 인용구 결합
	if firstParagraph != "" || quotationText != "" {
		combinedText := ""

		if firstParagraph != "" {
			combinedText = firstParagraph
		}

		if quotationText != "" {
			if combinedText != "" {
				combinedText += " "
			}
			combinedText += quotationText
		}

		result.FirstParagraph = combinedText
		fmt.Printf("최종 추출된 문단 (%d 바이트):\n%s\n", len(combinedText), combinedText)
	} else {
		fmt.Printf("문단과 인용구를 모두 찾지 못했습니다.\n")
	}
}

// cleanText는 텍스트에서 불필요한 문자를 제거합니다
func cleanText(text string) string {
	// 이모지, 특수문자 등 제거
	text = strings.ReplaceAll(text, "", "")       // 보이지 않는 공백 제거
	text = strings.ReplaceAll(text, "\u200b", "") // 제로 폭 공백 제거

	// sup 태그 텍스트 정리 (네이버 블로그에서 자주 사용됨)
	text = strings.ReplaceAll(text, "^", "")

	// 연속된 공백 정리
	regex := regexp.MustCompile(`\s+`)
	text = regex.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// min은 두 정수 중 작은 값을 반환합니다
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// containsStickerDomain은 URL이 스티커 도메인을 포함하는지 확인합니다
func containsStickerDomain(url string) bool {
	for _, domain := range constants.STICKER_DOMAINS {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

// parseTistoryBlog는 티스토리 블로그 HTML을 파싱합니다
func parseTistoryBlog(doc *goquery.Document, result *structure.CrawlResult) {
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
func parseGenericBlog(doc *goquery.Document, result *structure.CrawlResult) {
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
