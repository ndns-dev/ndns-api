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
	"reviewnote.co.kr",
	"xn--939au0g4vj8sq.net",
	"revu.net",
}

// 네이버 이미지 패턴
var NAVER_IMAGE_PATTERNS = []string{
	"blogfiles.naver.net",
	"postfiles.pstatic.net",
	"phinf.pstatic.net",
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

const MAX_IMAGE_SIZE = 12000000  // 1200만 픽셀 (약 4000x3000 크기)
const MAX_IMAGE_DIMENSION = 1200 // 픽셀 단위 (1200x1200 이상인 이미지는 크롭)
const CROP_HEIGHT = 500          // 상단 500픽셀만 사용
