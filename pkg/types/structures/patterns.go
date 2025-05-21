package structure

type PatternType string

const (
	PatternTypeSpecial PatternType = "special"
	PatternTypeExact   PatternType = "exact"
	PatternTypeNormal  PatternType = "normal"
)

// SpecialCasePattern은 특수 스폰서 패턴의 구조를 정의합니다
type SpecialCasePattern struct {
	Terms1 string
	Terms2 []string
}

// 1번 확인 패턴: SPECIAL_CASE_PATTERNS는 특수한 경우의 스폰서 패턴을 정의합니다
var SPECIAL_CASE_PATTERNS = []SpecialCasePattern{
	{
		Terms1: "업체",
		Terms2: []string{"지원", "원고료", "제공"},
	},
	{
		Terms1: "후기",
		Terms2: []string{"지원", "원고료", "제공"},
	},
	{
		Terms1: "댓가",
		Terms2: []string{"지원", "원고료", "제공"},
	},
	{
		Terms1: "서비스",
		Terms2: []string{"제공", "원고료", "작성"},
	},
	{
		Terms1: "광고",
		Terms2: []string{"콘텐츠", "원고료", "포스팅", "게시물"},
	},
}

// 2번 확인 패턴: EXACT_SPONSOR_KEYWORDS_PATTERNS는 정확한 스폰서 키워드 패턴을 정의합니다
var EXACT_SPONSOR_KEYWORDS_PATTERNS = []string{
	// 단어 일부 포함된 패턴
	"고료",
	"험단",

	"소정의",
	"협찬",
	"수수료",
	// ocr로 잘못 읽었지만 협찬 패턴
	"[현산",
	"현찬",
	"[.싫헐진",
}

// 스폰서 단일 키워드 (모호하고 일반적인 단어일수록 낮은 가중치)
var SPONSOR_KEYWORDS = map[string]float64{
	// 협찬 관련 키워드
	"체험":   0.3,
	"지원":   0.4,
	"무상":   0.4,
	"무료제공": 0.6,
	"제품제공": 0.7,
	// 유료 광고 관련 키워드
	"광고":   0.1,
	"광고비":  0.4,
	"유료광고": 0.8,
	// 제공 품목 관련 키워드
	"쿠폰":  0.4,
	"포인트": 0.4,
	"식사":  0.2,
	"이용권": 0.2,
	// 공통 키워드 (매우 낮은 가중치)
	"작성":    0.01,
	"후기":    0.01,
	"받았습니다": 0.2,
	"받아":    0.2,
	"받고":    0.2,
	"로부터":   0.1,
	"업체":    0.4,
	"혜택":    0.2,
	"제공":    0.4,
	"선정":    0.4,
	//ocr로 잘못 읽었지만 협찬 패턴
	"업제": 0.4,
	"입체": 0.4,
}

// 정확도
type SPONSOR_ACCURACY struct {
	Absolute  float64 // 확실한 협찬
	Exact     float64 // 정확한 협찬
	Possible  float64 // 가능한 협찬
	Ambiguous float64 // 모호한 협찬
}

var Accuracy = SPONSOR_ACCURACY{
	Absolute:  1.0,
	Exact:     0.9,
	Possible:  0.7,
	Ambiguous: 0.5,
}
