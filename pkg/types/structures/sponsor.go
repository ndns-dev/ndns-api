package structure

type SponsorIndicator struct {
	Type        IndicatorType `json:"type"`
	Pattern     PatternType   `json:"pattern"`
	MatchedText string        `json:"matchedText"`
	Probability float64       `json:"probability"`
	Source      SponsorSource `json:"source"`
}

type IndicatorType string

const (
	IndicatorTypeExactKeywordRegex IndicatorType = "exactKeywordRegex"
	IndicatorTypeKeyword           IndicatorType = "keyword"
)

// SponsorType은 협찬 유형을 정의합니다
type SponsorType int

const (
	SponsorTypeDescription    SponsorType = iota // 설명에서 발견
	SponsorTypeFirstParagraph                    // 첫 문단에서 발견
	SponsorTypeLastParagraph                     // 마지막 문단에서 발견
	SponsorTypeImage                             // 이미지에서 발견
	SponsorTypeSticker                           // 스티커에서 발견
	SponsorTypeUnknown                           // 알 수 없는 유형
)

type SponsorSource struct {
	SponsorType SponsorType `json:"sponsorType"`
	Text        string      `json:"text"`
}
