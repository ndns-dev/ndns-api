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

type CrawlResult struct {
	Url              string `json:"url"`
	FirstParagraph   string `json:"firstParagraph"`
	LastParagraph    string `json:"lastParagraph"`
	Content          string `json:"content"`
	FirstImageUrl    string `json:"firstImageUrl"`
	LastImageUrl     string `json:"lastImageUrl"`
	FirstStickerUrl  string `json:"firstStickerUrl"`
	SecondStickerUrl string `json:"secondStickerUrl"`
	LastStickerUrl   string `json:"lastStickerUrl"`
}
