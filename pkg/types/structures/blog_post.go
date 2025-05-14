package structure

type NaverSearchItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	BloggerName string `json:"bloggerName"`
	BloggerLink string `json:"bloggerLink"`
	PostDate    string `json:"postDate"`
}

type BlogImage string

const (
	BlogImageTypeImage   BlogImage = "image"
	BlogImageTypeSticker BlogImage = "sticker"
)

type NaverSearchResponse struct {
	LastBuildDate string            `json:"lastBuildDate"`
	Total         int               `json:"total"`
	Start         int               `json:"start"`
	Display       int               `json:"display"`
	Items         []NaverSearchItem `json:"items"`
}

type BlogPost struct {
	NaverSearchItem
	IsSponsored        bool               `json:"isSponsored"`
	SponsorProbability float64            `json:"sponsorProbability"`
	SponsorIndicators  []SponsorIndicator `json:"sponsorIndicators"`
}

type CrawlResult struct {
	URL             string
	FirstParagraph  string
	LastParagraph   string
	Content         string
	FirstImageURL   string
	LastImageURL    string
	FirstStickerURL string
	LastStickerURL  string
}
