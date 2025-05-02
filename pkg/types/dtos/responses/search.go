package response

import "github.com/sh5080/ndns-go/pkg/types/structures"

// SearchResponse는 검색 요청에 대한 응답을 나타냅니다.
type Search struct {
	Keyword          string                `json:"keyword"`
	TotalResults     int                   `json:"totalResults"`
	SponsoredResults int                   `json:"sponsoredResults"`
	Page             int                   `json:"page"`
	ItemsPerPage     int                   `json:"itemsPerPage"`
	Posts            []structures.BlogPost `json:"posts"`
}
