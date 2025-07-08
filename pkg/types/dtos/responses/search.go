package response

import structure "github.com/sh5080/ndns-go/pkg/types/structures"

// SearchResponse는 검색 요청에 대한 응답을 나타냅니다.
type Search struct {
	Keyword          string             `json:"keyword"`
	TotalResults     int                `json:"totalResults"`
	SponsoredResults int                `json:"sponsoredResults"`
	Page             int                `json:"page"`
	ItemsPerPage     int                `json:"itemsPerPage"`
	Posts            []AnalyzedResponse `json:"posts"`
}

type AnalyzeText struct {
	IsSponsored bool                         `json:"isSponsored"`
	Probability float64                      `json:"probability"`
	Indicators  []structure.SponsorIndicator `json:"indicators"`
}

type QueueSuccess struct {
	Status string `json:"status"`
}

type AnalyzedResponse struct {
	structure.NaverSearchItem
	IsSponsored        bool                         `json:"isSponsored"`
	SponsorProbability float64                      `json:"sponsorProbability"`
	SponsorIndicators  []structure.SponsorIndicator `json:"sponsorIndicators"`
	Error              string                       `json:"error,omitempty"`
}

type AnalyzeJobResponse struct {
	ReqId              string                     `json:"reqId"`
	JobId              string                     `json:"jobId"`
	IsSponsored        bool                       `json:"isSponsored"`
	SponsorProbability float64                    `json:"sponsorProbability"`
	SponsorIndicator   structure.SponsorIndicator `json:"sponsorIndicator"`
	Error              string                     `json:"error,omitempty"`
}
