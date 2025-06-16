package crawler

import (
	"encoding/json"
	"fmt"
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
// is2025OrLater가 true일 경우 마지막 데이터는 가져오지 않습니다.
func CrawlBlogPost(url string, is2025OrLater bool) (*structure.CrawlResult, error) {
	if url == "" {
		return nil, fmt.Errorf("URL이 비어 있습니다")
	}

	// URL 정규화
	url = normalizeURL(url)
	utils.DebugLog("크롤링 시작: %s (2025년 이후 포스트: %v)\n", url, is2025OrLater)

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
			// 2025년 이후 포스트 여부에 따라 다른 파싱 함수 호출
			if is2025OrLater {
				parseNaverBlogFirst(contentDoc, result)
			} else {
				parseNaverBlogFull(contentDoc, result)
			}
		}
	} else {
		return nil, fmt.Errorf("지원하지 않는 블로그 플랫폼입니다")
	}
	return result, nil
}

// fetchHTML은 URL에서 HTML을 가져와 goquery.Document로 반환합니다
func fetchHTML(url string) (*goquery.Document, error) {
	var (
		resp *http.Response
		err  error
	)

	client := &http.Client{
		Timeout: constants.TIMEOUT,
	}

	// 재시도 로직 구현
	for attempt := 0; attempt < constants.CRAWL_MAX_RETRIES; attempt++ {
		if attempt > 0 {
			time.Sleep(constants.CRAWL_RETRY_DELAY)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("요청 생성 실패: %v", err)
		}

		// User-Agent 설정
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

		resp, err = client.Do(req)
		if err == nil {
			break
		}

		if attempt == constants.CRAWL_MAX_RETRIES-1 {
			return nil, fmt.Errorf("요청 실행 실패 (%d번째 시도): %v", attempt+1, err)
		}
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	return doc, nil
}

// extractNaverIframeURL은 네이버 블로그 프레임셋에서 실제 콘텐츠 iframe URL을 추출합니다
func extractNaverIframeURL(doc *goquery.Document, originalURL string) string {
	iframeURL := ""
	doc.Find("iframe#mainFrame").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			iframeURL = src
		}
	})

	if iframeURL == "" {
		return originalURL
	}

	// 상대 경로를 절대 경로로 변환
	if !strings.HasPrefix(iframeURL, "http") {
		baseURL := "https://blog.naver.com"
		iframeURL = baseURL + iframeURL
	}

	return iframeURL
}

// parseNaverBlogFirst는 네이버 블로그 HTML에서 첫 번째 데이터만 파싱합니다 (2025년 이후)
func parseNaverBlogFirst(doc *goquery.Document, result *structure.CrawlResult) {
	// 첫 번째 스티커 이미지 추출
	extractFirstStickerOnly(doc, result)
	// 첫 번째 일반 이미지 추출
	extractFirstImageOnly(doc, result)
	// 첫 번째 문단 추출
	extractFirstParagraphOnly(doc, result)
}

// parseNaverBlogFull은 네이버 블로그 HTML에서 모든 데이터를 파싱합니다 (2025년 이전)
func parseNaverBlogFull(doc *goquery.Document, result *structure.CrawlResult) {
	// 첫 번째 스티커 이미지 추출
	extractFirstSticker(doc, result)
	// 일반 이미지 추출
	extractFirstImage(doc, result)
	// 첫 문단 추출
	extractFirstParagraph(doc, result)
}

// extractFirstStickerOnly는 첫 번째 스티커만 추출합니다 (2025년 이후 포스트용)
func extractFirstStickerOnly(doc *goquery.Document, result *structure.CrawlResult) {
	// 모든 스티커 URL을 저장할 슬라이스
	var stickerURLs []string

	// 1. 스티커 클래스로 찾기
	for _, stickerClass := range constants.STICKER_CLASSES {
		doc.Find("[class*='" + stickerClass + "']").Each(func(i int, elem *goquery.Selection) {
			// 이미지 태그 찾기
			img := elem.Find("img")
			if img.Length() > 0 && img.AttrOr("src", "") != "" {
				imgURL := img.AttrOr("src", "")
				// 스티커 도메인 확인
				for _, domain := range constants.STICKER_DOMAINS {
					if strings.Contains(imgURL, domain) {
						stickerURLs = append(stickerURLs, imgURL)
						break
					}
				}
			}

			// 첫 번째 스티커를 찾았으면 루프 종료
			if len(stickerURLs) > 0 {
				return
			}
		})

		if len(stickerURLs) > 0 {
			break
		}
	}

	// 2. data-linkdata 속성으로 찾기 (네이버 블로그 특유의 구조)
	if len(stickerURLs) == 0 {
		doc.Find("[data-linkdata]").Each(func(i int, elem *goquery.Selection) {
			linkData := elem.AttrOr("data-linkdata", "")
			if linkData != "" {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(linkData), &data)
				if err == nil && data["src"] != nil {
					imgURL := data["src"].(string)
					// 스티커 도메인 확인
					for _, domain := range constants.STICKER_DOMAINS {
						if strings.Contains(imgURL, domain) {
							stickerURLs = append(stickerURLs, imgURL)
							break
						}
					}
				}
			}

			// 첫 번째 스티커를 찾았으면 루프 종료
			if len(stickerURLs) > 0 {
				return
			}
		})
	}

	// 결과 설정
	if len(stickerURLs) > 0 {
		result.FirstStickerURL = stickerURLs[0]
		if len(stickerURLs) > 1 {
			// 두 번째 스티커 URL 저장
			result.SecondStickerURL = stickerURLs[1]
			// 마지막 스티커는 수집하지 않음 (2025년 이후 포스트)
			result.LastStickerURL = ""
		} else {
			result.SecondStickerURL = ""
			result.LastStickerURL = ""
		}
	}
}

// extractFirstImageOnly는 첫 번째 이미지만 추출합니다 (2025년 이후 포스트용)
func extractFirstImageOnly(doc *goquery.Document, result *structure.CrawlResult) {
	// 모든 이미지 URL을 저장할 슬라이스
	var imageURLs []string

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

	// 이미지 검색 (첫 번째 이미지만 찾음)
	contentArea.Find("img").Each(func(i int, img *goquery.Selection) {
		// 상위 요소가 스티커 관련 요소가 아닌지 확인
		if img.ParentsFiltered("[class*='sticker']").Length() > 0 {
			return // 스티커 관련 요소 내부의 이미지는 건너뜀
		}

		imgURL := img.AttrOr("src", "")
		if imgURL == "" {
			imgURL = img.AttrOr("data-src", "")
		}

		if imgURL != "" {
			// 이미지가 스티커가 아닌지 확인
			isSticker := false
			for _, domain := range constants.STICKER_DOMAINS {
				if strings.Contains(imgURL, domain) {
					isSticker = true
					break
				}
			}

			// 제외 패턴에 포함된 이미지인지 확인 ex) 네이버 지도 이미지
			isExcluded := false
			for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
				if strings.Contains(imgURL, pattern) {
					isExcluded = true
					break
				}
			}

			// 스티커가 아니고 제외 패턴에 포함되지 않은 이미지만 추가
			if !isSticker && !isExcluded {
				imageURLs = append(imageURLs, imgURL)
				return // 첫 번째 이미지를 찾았으면 루프 종료
			}
		}
	})

	// 결과 설정
	if len(imageURLs) > 0 {
		result.FirstImageURL = imageURLs[0]
		// 마지막 이미지는 수집하지 않음 (2025년 이후 포스트)
		result.LastImageURL = ""
	}
	if strings.HasSuffix(result.FirstImageURL, "w80_blur") {
		result.FirstImageURL = strings.Replace(result.FirstImageURL, "w80_blur", "w773", 1)
	}
}

// extractCommonParagraphs는 HTML 문서에서 문단을 추출하는 공통 함수입니다
// maxParagraphs가 양수일 경우 처음부터 최대 maxParagraphs개만 가져오고
// maxParagraphs가 0일 경우 모든 문단을 가져옵니다
func extractCommonParagraphs(doc *goquery.Document, maxParagraphs int) []string {
	// 모든 문단을 저장할 슬라이스
	var paragraphs []string

	// 본문 영역 찾기
	var contentArea *goquery.Selection
	for _, selector := range constants.CONTENT_SELECTORS {
		selected := doc.Find(selector)
		if selected.Length() > 0 {
			contentArea = selected.First()
			break
		}
	}

	// 본문 영역을 찾지 못한 경우 전체 HTML 사용
	if contentArea == nil || contentArea.Length() == 0 {
		contentArea = doc.Selection
	}

	// 문단 선택자 확인
	paragraphSelectors := []string{
		".se-text-paragraph", // 스마트에디터 2.0 문단
		".se-module-text p",  // 스마트에디터 모듈 내 문단
		".post_ct p",         // 일반 모바일 블로그 문단
		".sect_dsc p",        // 모바일 본문 문단
		"p",                  // 일반 문단 태그
	}

	// 문단 찾기
	for _, selector := range paragraphSelectors {
		if maxParagraphs > 0 && len(paragraphs) >= maxParagraphs {
			break
		}

		paragraphElements := contentArea.Find(selector)
		paragraphElements.Each(func(i int, p *goquery.Selection) {
			if maxParagraphs > 0 && len(paragraphs) >= maxParagraphs {
				return
			}

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

			if text != "" && len(text) > 5 {
				paragraphs = append(paragraphs, text)
			}
		})

		// 첫 번째 선택자에서 일치하는 문단을 찾으면 다른 선택자는 확인하지 않음
		// 그러나 모든 문단을 가져오는 경우는 계속 진행
		if len(paragraphs) > 0 && maxParagraphs > 0 && maxParagraphs <= 3 {
			break
		}
	}

	// 문단을 찾지 못한 경우 대체 방법 시도
	if len(paragraphs) == 0 {
		// div 직접 검색
		contentArea.Find("div.se-module-text").Each(func(i int, div *goquery.Selection) {
			if i > 10 || (maxParagraphs > 0 && len(paragraphs) >= maxParagraphs) { // 처음 10개만 확인
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
				paragraphs = append(paragraphs, text)
			}
		})
	}

	return paragraphs
}

// extractLastParagraphs는 주어진 문단 배열에서 마지막 maxParagraphs개의 문단을 반환합니다
func extractLastParagraphs(paragraphs []string, maxParagraphs int) []string {
	if len(paragraphs) == 0 {
		return []string{}
	}

	startIndex := max(0, len(paragraphs)-maxParagraphs)
	return paragraphs[startIndex:]
}

// extractFirstParagraphOnly는 첫 번째 문단만 추출합니다 (2025년 이후 포스트용)
func extractFirstParagraphOnly(doc *goquery.Document, result *structure.CrawlResult) {
	// 최대 3개의 문단 추출
	paragraphs := extractCommonParagraphs(doc, 10)

	// FirstParagraph 설정 - 문단 병합
	if len(paragraphs) > 0 {
		result.FirstParagraph = strings.Join(paragraphs, " ")
		// 마지막 문단은 수집하지 않음 (2025년 이후 포스트)
		result.LastParagraph = ""
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

// normalizeURL은 URL을 정규화합니다
func normalizeURL(url string) string {
	// HTTP/HTTPS 접두사 추가
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return url
}

// extractFirstSticker는 첫 번째와 마지막 스티커를 추출합니다
func extractFirstSticker(doc *goquery.Document, result *structure.CrawlResult) {
	// 모든 스티커 URL을 저장할 슬라이스
	var stickerURLs []string

	// 1. 스티커 클래스로 찾기
	for _, stickerClass := range constants.STICKER_CLASSES {
		doc.Find("[class*='" + stickerClass + "']").Each(func(i int, elem *goquery.Selection) {
			// 이미지 태그 찾기
			img := elem.Find("img")
			if img.Length() > 0 && img.AttrOr("src", "") != "" {
				imgURL := img.AttrOr("src", "")
				// 스티커 도메인 확인
				for _, domain := range constants.STICKER_DOMAINS {
					if strings.Contains(imgURL, domain) {
						stickerURLs = append(stickerURLs, imgURL)
						break
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
							stickerURLs = append(stickerURLs, imgURL)
							break
						}
					}
				}
			}
		})

		if len(stickerURLs) > 0 {
			break
		}
	}

	// 2. data-linkdata 속성으로 찾기 (네이버 블로그 특유의 구조)
	if len(stickerURLs) == 0 {
		doc.Find("[data-linkdata]").Each(func(i int, elem *goquery.Selection) {
			linkData := elem.AttrOr("data-linkdata", "")
			if linkData != "" {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(linkData), &data)
				if err == nil && data["src"] != nil {
					imgURL := data["src"].(string)
					// 스티커 도메인 확인
					for _, domain := range constants.STICKER_DOMAINS {
						if strings.Contains(imgURL, domain) {
							stickerURLs = append(stickerURLs, imgURL)
							break
						}
					}
				}
			}
		})
	}

	// 3. 이미지 태그 확인
	if len(stickerURLs) == 0 {
		doc.Find("img").Each(func(i int, img *goquery.Selection) {
			imgURL := img.AttrOr("src", "")
			// 스티커 도메인 확인
			for _, domain := range constants.STICKER_DOMAINS {
				if strings.Contains(imgURL, domain) {
					stickerURLs = append(stickerURLs, imgURL)
					break
				}
			}
		})
	}
	// 결과 설정
	if len(stickerURLs) > 0 {
		result.FirstStickerURL = stickerURLs[0]
		if len(stickerURLs) > 1 {
			// 두 번째 스티커 URL 저장
			result.SecondStickerURL = stickerURLs[1]
			// 마지막 스티커는 수집하지 않음 (2025년 이후 포스트)
			result.LastStickerURL = stickerURLs[len(stickerURLs)-1]
		} else {
			result.SecondStickerURL = ""
			result.LastStickerURL = ""
		}
	}
}

// extractFirstImage는 본문 영역에서 첫 번째와 마지막 이미지를 추출합니다
func extractFirstImage(doc *goquery.Document, result *structure.CrawlResult) {
	// 모든 이미지 URL을 저장할 슬라이스
	var imageURLs []string

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

	// 0. 인라인 이미지 및 data-linkdata 속성 확인 (협찬 이미지에 특히 중요)
	contentArea.Find(".se-inline-image, .se-module-image").Each(func(i int, imgContainer *goquery.Selection) {
		// data-linkdata 속성 확인 (JSON 데이터로 이미지 URL 포함)
		linkElem := imgContainer.Find("a[data-linkdata], [data-linkdata]").First()
		if linkElem.Length() > 0 {
			linkData := linkElem.AttrOr("data-linkdata", "")
			if linkData != "" {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(linkData), &data)
				if err == nil && data["src"] != nil {
					imgURL := data["src"].(string)

					// 제외 패턴 확인
					isExcluded := false
					for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
						if strings.Contains(imgURL, pattern) {
							isExcluded = true
							break
						}
					}

					// 제외되지 않은 이미지만 추가
					if !isExcluded {
						imageURLs = append(imageURLs, imgURL)
					}
				}
			}
		}

		// 직접 이미지 태그 확인
		img := imgContainer.Find("img").First()
		if img.Length() > 0 {
			imgURL := img.AttrOr("src", "")
			if imgURL != "" {
				// 제외 패턴 확인
				isExcluded := false
				for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
					if strings.Contains(imgURL, pattern) {
						isExcluded = true
						break
					}
				}

				// 제외되지 않은 이미지만 추가
				if !isExcluded {
					imageURLs = append(imageURLs, imgURL)
				}
			}
		}
	})

	// 이미지를 이미 찾았으면 다른 방법은 계속 진행

	// 1. 스마트에디터 이미지 리소스 확인
	contentArea.Find(".se-image-resource").Each(func(i int, img *goquery.Selection) {
		// 상위 요소가 스티커 모듈이 아닌지 확인
		if img.ParentsFiltered(".se-module-sticker").Length() > 0 {
			return // 스티커 모듈 내부의 이미지는 건너뜀
		}

		imgURL := img.AttrOr("src", "")
		if imgURL != "" {
			// 제외 패턴 확인
			isExcluded := false
			for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
				if strings.Contains(imgURL, pattern) {
					isExcluded = true
					break
				}
			}

			// 제외되지 않은 이미지만 추가
			if !isExcluded {
				imageURLs = append(imageURLs, imgURL)
			}
		}
	})

	// 2. se-component 이미지 확인
	contentArea.Find(".se-component.se-image").Each(func(i int, component *goquery.Selection) {
		// 이미지 모듈 찾기
		img := component.Find(".se-module-image .se-image-resource").First()
		if img.Length() > 0 {
			imgURL := img.AttrOr("src", "")
			if imgURL != "" {
				// 제외 패턴 확인
				isExcluded := false
				for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
					if strings.Contains(imgURL, pattern) {
						isExcluded = true
						break
					}
				}

				// 제외되지 않은 이미지만 추가
				if !isExcluded {
					imageURLs = append(imageURLs, imgURL)
					return
				}
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

					// 제외 패턴 확인
					isExcluded := false
					for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
						if strings.Contains(imgURL, pattern) {
							isExcluded = true
							break
						}
					}

					// 제외되지 않은 이미지만 추가
					if !isExcluded {
						imageURLs = append(imageURLs, imgURL)
					}
				}
			}
		}
	})

	// 3. 일반 이미지 확인
	contentArea.Find("img").Each(func(i int, img *goquery.Selection) {
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

		if imgURL != "" && (strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://")) {
			// 제외 패턴 확인
			isExcluded := false
			for _, pattern := range constants.EXCLUDE_IMAGE_PATTERNS {
				if strings.Contains(imgURL, pattern) {
					isExcluded = true
					break
				}
			}

			// 제외되지 않은 이미지만 추가
			if !isExcluded {
				imageURLs = append(imageURLs, imgURL)
			}
		}
	})

	// 결과 설정
	if len(imageURLs) > 0 {
		result.FirstImageURL = imageURLs[0]
		if strings.HasSuffix(result.FirstImageURL, "w80_blur") {
			result.FirstImageURL = strings.Replace(result.FirstImageURL, "w80_blur", "w773", 1)
		}

		if len(imageURLs) > 1 {
			result.LastImageURL = imageURLs[len(imageURLs)-1]
			if strings.HasSuffix(result.LastImageURL, "w80_blur") {
				result.LastImageURL = strings.Replace(result.LastImageURL, "w80_blur", "w773", 1)
			}
		} else {
			result.LastImageURL = result.FirstImageURL
		}
	}
}

// extractFirstParagraph는 첫 번째와 마지막 문단과 인용구를 추출합니다
func extractFirstParagraph(doc *goquery.Document, result *structure.CrawlResult) {
	// 모든 문단 추출
	paragraphs := extractCommonParagraphs(doc, 0) // 0은 모든 문단을 의미

	// 인용구 확인
	var quotations []string

	// 인용구 선택자 확인 (네이버 블로그 스마트에디터 패턴에 맞게 개선)
	quotationSelectors := []string{
		".se-quotation-container", // 스마트에디터 2.0 인용구
		"blockquote",              // 일반 인용구
	}

	var contentArea *goquery.Selection
	for _, selector := range constants.CONTENT_SELECTORS {
		selected := doc.Find(selector)
		if selected.Length() > 0 {
			contentArea = selected.First()
			break
		}
	}

	// 본문 영역을 찾지 못한 경우 전체 HTML 사용
	if contentArea == nil || contentArea.Length() == 0 {
		contentArea = doc.Selection
	}

	for _, selector := range quotationSelectors {
		quotes := contentArea.Find(selector)

		if quotes.Length() > 0 {
			quotes.Each(func(i int, quote *goquery.Selection) {
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
					quotations = append(quotations, text)
				}
			})
		}

		if len(quotations) > 0 {
			break
		}
	}

	// 인용구와 문단 결합
	var allTexts []string
	allTexts = append(allTexts, paragraphs...)
	allTexts = append(allTexts, quotations...)

	// FirstParagraph 설정 - 첫 3개 문단
	if len(allTexts) > 0 {
		// 첫 번째 문단 - 최대 3개의 문단 병합
		firstParagraphs := allTexts
		if len(allTexts) > 3 {
			firstParagraphs = allTexts[:3]
		}
		result.FirstParagraph = strings.Join(firstParagraphs, " ")

		// LastParagraph 설정 - 마지막 3개 문단
		if len(allTexts) > 1 {
			lastParagraphs := extractLastParagraphs(allTexts, 3)
			result.LastParagraph = strings.Join(lastParagraphs, " ")
		} else {
			result.LastParagraph = result.FirstParagraph
		}
	} else {
		fmt.Printf("문단과 인용구를 모두 찾지 못했습니다.\n")
	}
}
