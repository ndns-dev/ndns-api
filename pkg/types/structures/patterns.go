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
	{
		Terms1: "로부터",
		Terms2: []string{"업체", "작성", "하였", "받았"},
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
	// 협찬 서비스 패턴
	"슈퍼멤버스",
	"대세블",
	"대서블",
	// ocr로 잘못 읽었지만 협찬 패턴
	"협잔",
	"협깐",
	"[현산",
	"현찬",
	"[.싫헐진",
}

// 스폰서 단일 키워드 (모호하고 일반적인 단어일수록 낮은 가중치)
var SPONSOR_KEYWORDS = map[string]float64{
	// 협찬 관련 키워드
	"체험":   0.5,
	"지원":   0.3,
	"무상":   0.4,
	"무료제공": 0.6,
	"제품제공": 0.7,
	// 유료 광고 관련 키워드
	"광고":  0.1,
	"광고비": 0.4,
	"유료":  0.5,
	// 제공 품목 관련 키워드
	"쿠폰":  0.4,
	"포인트": 0.4,
	"식사":  0.2,
	"이용권": 0.2,
	// 공통 키워드 (매우 낮은 가중치)
	"작":  0.05,
	"작성": 0.1,
	"후기": 0.05,
	"받아": 0.2,
	"받고": 0.2,
	"로부": 0.1, // 로부터
	"받았": 0.3,
	"혜택": 0.2,
	"솔직": 0.2,
	"리뷰": 0.3,
	"포함": 0.2,
	// 포스트, 포스팅 등
	"포스": 0.1,
	// 협찬 관련 키워드
	"업체": 0.45,
	"제공": 0.4,
	"선정": 0.4,
	//ocr로 잘못 읽었지만 협찬 패턴
	"팡고": 0.1,
	"유로": 0.2, // 유료 잘못읽을 수 있는 패턴, 실제 유로 일 수 있어 낮은 가중치로 둠
	"업제": 0.45,
	"입체": 0.45,
	"제험": 0.5,
	"쳐혐": 0.5,
	"쳐험": 0.5,
	"스트": 0.1,
	"스팅": 0.1,
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
