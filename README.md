# NDNS-GO (내돈내산 필터링 서비스)

## 소개

NDNS-GO는 네이버 검색 API를 통해 블로그에 키워드를 검색하면 해당 포스트가 협찬을 받은 것인지 필터링하는 서비스입니다. 
본 프로젝트는 Go 언어와 Fiber 프레임워크를 사용하여 개발되었으며, 기존 Python 구현체의 성능 및 동시성 문제를 해결하기 위해 재구현되었습니다.

## 주요 기능

- 네이버 검색 API를 통한 블로그 포스트 검색
- 검색된 블로그 포스트의 협찬 여부 탐지
  - 텍스트 분석 (첫 1-2개 문단, 인용구 등)
  - 이미지 분석 (OCR을 통한 텍스트 추출)
  - 스티커 분석 (OCR을 통한 텍스트 추출)
- DynamoDB를 활용한 OCR 결과 캐싱
- 병렬 처리를 통한 높은 성능 및 동시성 지원

## 기술 스택

- 언어: Go
- 웹 프레임워크: Fiber
- HTML 파싱: goquery
- OCR: 외부 OCR API (구현 필요)
- 캐싱: AWS DynamoDB

## 설치 및 실행

### 사전 요구사항

- Go 1.16 이상
- AWS 계정 (DynamoDB 사용 시)
- 네이버 검색 API 클라이언트 ID와 시크릿

### 환경 변수 설정

```
NAVER_CLIENT_ID=네이버_API_클라이언트_ID
NAVER_CLIENT_SECRET=네이버_API_클라이언트_시크릿
DYNAMODB_REGION=ap-northeast-2
DYNAMODB_TABLE_NAME=ocr-cache
PORT=8080
```

### 빌드 및 실행

```bash
# 의존성 설치
go mod download

# 빌드
go build -o ndns-go ./pkg

# 실행
./ndns-go
```

## API 사용법

### 블로그 포스트 검색 및 협찬 필터링

```
GET /api/search/naver?query={검색어}
```

#### 응답 예시

```json
{
  "success": true,
  "results": [
    {
      "post": {
        "title": "맛있는 식당 방문기",
        "link": "https://blog.naver.com/user/12345",
        "description": "오늘은 맛있는 식당에 다녀왔어요.",
        "bloggerName": "맛집탐방러",
        "bloggerLink": "https://blog.naver.com/user",
        "postDate": "2023-06-15T00:00:00Z"
      },
      "isSponsored": true,
      "confidence": 0.8,
      "detectionMethod": "텍스트 분석",
      "keywordsFound": ["제공", "체험"],
      "patternMatches": {
        "업체 + 지원/제공": true
      }
    },
    {
      "post": {
        "title": "나의 솔직한 후기",
        "link": "https://blog.naver.com/user/67890",
        "description": "정직한 내 돈 내산 리뷰입니다.",
        "bloggerName": "솔직리뷰어",
        "bloggerLink": "https://blog.naver.com/user2",
        "postDate": "2023-06-10T00:00:00Z"
      },
      "isSponsored": false,
      "confidence": 0.1,
      "detectionMethod": "텍스트 분석",
      "keywordsFound": []
    }
  ]
}
```

## 프로젝트 구조

```
.
├── api/
│   ├── controllers/   # API 엔드포인트 핸들러
│   ├── models/        # 데이터 모델
│   ├── services/      # 비즈니스 로직
│   └── utils/         # 유틸리티 함수
├── configs/           # 설정 관리
├── main.go            # 애플리케이션 진입점
└── README.md          # 프로젝트 설명
```

## 기여 방법

1. 이 저장소를 포크합니다.
2. 새로운 기능 브랜치를 생성합니다: `git checkout -b feature/amazing-feature`
3. 변경사항을 커밋합니다: `git commit -m 'Add some amazing feature'`
4. 브랜치에 푸시합니다: `git push origin feature/amazing-feature`
5. Pull Request를 제출합니다.

## 라이선스

이 프로젝트는 MIT 라이선스를 따릅니다. 