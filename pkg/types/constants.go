package constants

import "time"

// 스티커 도메인 패턴
var STICKER_DOMAINS = []string{
	"storep-phinf.pstatic.net",
	"post-phinf.pstatic.net",
}

// 협찬 업체 도메인 패턴
var SPONSOR_DOMAINS = []string{
	"cometoplay.kr",
	"xn--939au0g4vj8sq.net",
	"revu.net",
	"storyn.kr",
	"dinnerqueen.net",
	"review",
	"ringble",
	"강남맛집",
	"모두모여",
	"mrble1",
}

// 네이버 이미지 패턴
var NAVER_IMAGE_PATTERNS = []string{
	"blogfiles.naver.net",
	"postfiles.pstatic.net",
	"phinf.pstatic.net",
}

// 이미지에서 제외 패턴
var EXCLUDE_IMAGE_PATTERNS = []string{
	"simg.pstatic.net", // 네이버 지도 이미지
}

// 스티커 클래스 패턴
var STICKER_CLASSES = []string{
	"se-sticker",
	"sticker",
	"_img",
	"sponsor-tag",
	"ad-tag",
	"se-module",
	"se-module-image",
	"se-image-resource",
}

// 콘텐츠 영역 선택자
var CONTENT_SELECTORS = []string{
	".se-main-container", // 스마트에디터 2.0
	".post_ct",           // 구버전 모바일
	"#viewTypeSelector",  // 구버전 PC
	".se_component_wrap", // 구버전 PC (스마트에디터 1.0)
	".se-module-text",    // 텍스트 모듈
	".sect_dsc",          // 모바일 본문
	".se_card_container", // 카드 컨테이너
	"#postViewArea",      // 일반 포스트
	".post-content",      // 일반적인 블로그 본문 클래스
}

// 타임아웃 시간
var TIMEOUT = 4 * time.Second

// 크롤링 설정
const (
	CRAWL_MAX_RETRIES = 3
	CRAWL_RETRY_DELAY = 500 * time.Millisecond
)
