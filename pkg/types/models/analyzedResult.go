package model

import (
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

type AnalyzedResult struct {
	Link               string                       `json:"link" dynamodbav:"link"`
	IsSponsored        bool                         `json:"isSponsored" dynamodbav:"isSponsored"`
	SponsorProbability float64                      `json:"sponsorProbability" dynamodbav:"sponsorProbability"`
	SponsorIndicators  []structure.SponsorIndicator `json:"sponsorIndicators" dynamodbav:"sponsorIndicators"`
}
