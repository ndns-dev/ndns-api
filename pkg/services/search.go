package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sh5080/ndns-go/pkg/types/structures"
)

// NewSearchService는 새 검색 서비스를 생성합니다
func NewSearchService() SearchService {
	return &SearchImpl{
		crawlerService: NewCrawlerService(),
		ocrService:     NewOCRService(),
	}
}

// SearchBlogPosts는 검색어로 블로그 포스트를 검색합니다
func (s *SearchImpl) SearchBlogPosts(query string, count int, start int) ([]structures.BlogPost, error) {

	// 네이버 검색 API URL 구성
	searchURL := s.config.Naver.SearchURL

	// URL 파라미터 추가
	params := url.Values{}
	params.Add("query", query)
	params.Add("display", fmt.Sprintf("%d", count))
	params.Add("start", fmt.Sprintf("%d", start))
	params.Add("sort", "sim") // 정확도순 정렬

	// 요청 URL 생성
	reqURL := searchURL + "?" + params.Encode()

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %v", err)
	}

	// API 인증 헤더 추가
	req.Header.Add("X-Naver-Client-Id", s.config.Naver.ClientID)
	req.Header.Add("X-Naver-Client-Secret", s.config.Naver.ClientSecret)

	// 요청 실행
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, string(body))
	}

	// 응답 JSON 파싱
	var naverResp structures.NaverSearchResponse
	if err := json.Unmarshal(body, &naverResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}

	posts, err := s.DetectSponsor(naverResp.Items)
	if err != nil {
		return nil, fmt.Errorf("스폰서 감지 실패: %v", err)
	}

	return posts, nil
}
