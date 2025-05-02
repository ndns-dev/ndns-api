package request

// SearchQuery는 검색 요청 쿼리를 나타냅니다.
type SearchQuery struct {
	Query  string `json:"query" validate:"required,min=2,max=100"`
	Limit  int    `json:"limit,omitempty" validate:"min=1,max=100"`
	Offset int    `json:"offset,omitempty" validate:"min=0"`
}
