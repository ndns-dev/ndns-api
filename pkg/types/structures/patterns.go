package structure

var EXACT_SPONSOR_KEYWORDS_PATTERNS = []string{
	"원고료",
	"소정의",
	"체험단",
	"협찬",
	// ocr로 잘못 읽었지만 협찬 패턴
	"[현산",
	"현찬",
	"[.싫헐진",
}

// SpecialCasePattern은 특수 스폰서 패턴의 구조를 정의합니다
type SpecialCasePattern struct {
	Terms1 []string
	Terms2 []string
}

// SPECIAL_CASE_PATTERNS는 특수한 경우의 스폰서 패턴을 정의합니다
var SPECIAL_CASE_PATTERNS = map[string]SpecialCasePattern{
	"업체 + 지원/제공": {
		Terms1: []string{"업체"},
		Terms2: []string{"지원", "제공"},
	},
	"후기 + 지원/제공": {
		Terms1: []string{"후기"},
		Terms2: []string{"지원", "제공"},
	},
	"광고 + 콘텐츠": {
		Terms1: []string{"광고"},
		Terms2: []string{"콘텐츠", "포스팅", "게시물"},
	},
	"AD + 포스팅": {
		Terms1: []string{"ad"},
		Terms2: []string{"포스팅", "콘텐츠", "게시물"},
	},
}

// 스폰서 단일 키워드 (모호하고 일반적인 단어일수록 낮은 가중치)
var SPONSOR_KEYWORDS = map[string]float64{
	// 협찬 관련 키워드
	// "협찬": 0.8,
	"체험단":  0.6,
	"체험":   0.3,
	"지원":   0.4,
	"제공":   0.4,
	"무상":   0.4,
	"무료제공": 0.6,
	// EXACT_SPONSOR_KEYWORDS_PATTERNS 에 있어서 제외함
	// "원고료": 0.9,
	// "소정의": 0.9,
	"고료":   0.6,
	"제품제공": 0.7,
	// 유료 광고 관련 키워드
	"광고":   0.01,
	"유료광고": 0.8,
	// 공통 키워드 (매우 낮은 가중치)
	"작성":    0.01,
	"후기":    0.01,
	"받았습니다": 0.2,
	"받아":    0.01,
	"받고":    0.01,
	"로부터":   0.01,
	"업체":    0.4,
	"업제":    0.4,
}
