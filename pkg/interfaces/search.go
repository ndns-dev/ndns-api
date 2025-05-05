package _interface

import (
	request "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// SearchService는 검색 서비스 인터페이스입니다
type SearchService interface {
	// SearchBlogPosts는 검색어로 블로그 포스트를 검색합니다
	SearchBlogPosts(req request.SearchQuery) ([]structure.BlogPost, error)
}

// SponsorService는 스폰서 감지 서비스 인터페이스입니다
type SponsorService interface {
	// DetectSponsor는 블로그 포스트에서 스폰서를 감지합니다
	DetectSponsor(posts []structure.NaverSearchItem) ([]structure.BlogPost, error)
}

// CrawlerService는 블로그 콘텐츠를 크롤링하는 인터페이스입니다
type CrawlerService interface {
	// CrawlBlogPost는 블로그 포스트 URL에서 콘텐츠를 크롤링합니다
	CrawlBlogPost(url string) (*structure.CrawlResult, error)
}
