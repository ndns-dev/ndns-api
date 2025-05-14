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
type SponsorType string

const (
	SponsorTypeDescription SponsorType = "description" // 설명에서 발견
	SponsorTypeParagraph   SponsorType = "paragraph"   // 첫 문단에서 발견
	SponsorTypeImage       SponsorType = "image"       // 이미지에서 발견
	SponsorTypeSticker     SponsorType = "sticker"     // 스티커에서 발견
	SponsorTypeUnknown     SponsorType = "unknown"     // 알 수 없는 유형
)

type SponsorSource struct {
	SponsorType SponsorType `json:"sponsorType"`
	Text        string      `json:"text"`
}
