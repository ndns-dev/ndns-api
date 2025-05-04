package structure

type SponsorIndicator struct {
	Type        IndicatorType  `json:"type"`
	Pattern     string         `json:"pattern"`
	MatchedText string         `json:"matchedText"`
	Probability float64        `json:"probability"`
	Source      *SponsorSource `json:"source"`
}

type IndicatorType string

const (
	IndicatorTypeExactKeywordRegex IndicatorType = "exactKeywordRegex"
	IndicatorTypeKeyword           IndicatorType = "keyword"
)

type SponsorType string

const (
	SponsorTypeFirstParagraph SponsorType = "firstParagraph"
	SponsorTypeImage          SponsorType = "image"
	SponsorTypeSticker        SponsorType = "sticker"
)

type SponsorSource struct {
	SponsorType SponsorType `json:"sponsorType"`
	Text        string      `json:"text"`
}
