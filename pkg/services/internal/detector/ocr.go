package detector

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	constants "github.com/sh5080/ndns-go/pkg/types"
)

// OCRImpl는 OCR 서비스 구현체입니다
type OCRImpl struct {
	_interface.Service
	ocrRepo _interface.OCRRepository
}

// NewOCRService는 새 OCR 서비스를 생성합니다
func NewOCRService() _interface.OCRService {
	return &OCRImpl{
		Service: _interface.Service{
			Client: &http.Client{
				Timeout: time.Second * 30,
			},
			Config: configs.GetConfig(),
		},
		ocrRepo: repository.NewOCRRepository(),
	}
}

// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
func (o *OCRImpl) ExtractTextFromImage(imageURL string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	// 캐시 확인
	if o.ocrRepo != nil {
		cache, err := o.ocrRepo.GetOCRCache(imageURL)
		if err == nil && cache != nil && cache.TextDetected != "" {
			return cache.TextDetected, nil
		}
	}

	// 이미지 다운로드
	tempFile, err := o.downloadImage(imageURL)
	if err != nil {
		return "", fmt.Errorf("이미지 다운로드 실패: %v", err)
	}
	defer os.Remove(tempFile) // 임시 파일 정리

	// OCR 실행
	textDetected, err := o.runOCR(tempFile)
	if err != nil {
		fmt.Printf("OCR 실행 실패: %v, 파일 경로: %s\n", err, tempFile)
		return "", fmt.Errorf("OCR 실행 실패: %v", err)
	}

	// OCR 결과 캐싱
	if o.ocrRepo != nil && textDetected != "" {
		// 비동기 저장 (결과에 영향 없음)
		go func() {
			_ = o.ocrRepo.SaveOCRCache(imageURL, textDetected, "image")
		}()
	}

	return textDetected, nil
}

// downloadImage는 이미지 URL에서 이미지를 다운로드합니다
func (o *OCRImpl) downloadImage(imageURL string) (string, error) {
	// URL 정규화
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageURL = "https://" + imageURL
	}

	// URL 인코딩: 한글 등 특수 문자가 포함된 경우 인코딩
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return "", fmt.Errorf("URL 파싱 오류: %v", err)
	}

	// 각 경로 세그먼트를 개별적으로 인코딩
	pathSegments := strings.Split(parsedURL.Path, "/")
	for i, segment := range pathSegments {
		if segment != "" {
			pathSegments[i] = url.PathEscape(segment)
		}
	}
	parsedURL.Path = strings.Join(pathSegments, "/")

	// 인코딩된 URL로 업데이트
	imageURL = parsedURL.String()

	// URL 최적화: 네이버 블로그 이미지 크기 조정
	// ?type= 매개변수가 없는 경우 w773 크기 추가
	isNaverImage := false

	// 네이버 이미지 패턴 확인
	for _, pattern := range constants.NAVER_IMAGE_PATTERNS {
		if strings.Contains(imageURL, pattern) {
			isNaverImage = true
			break
		}
	}

	if isNaverImage {
		if !strings.Contains(imageURL, "?type=") && !strings.Contains(imageURL, "&type=") {
			// URL에 이미 쿼리 파라미터가 있는지 확인
			if strings.Contains(imageURL, "?") {
				imageURL += "&type=w773" // 쿼리 매개변수가 이미 있으면 &로 추가
			} else {
				imageURL += "?type=w773" // 쿼리 매개변수가 없으면 ?로 시작
			}
		}
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 요청 헤더 추가 (브라우저 에뮬레이션)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Add("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Add("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

	// 요청 실행
	resp, err := o.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 오류 (%d)", resp.StatusCode)
	}

	// 임시 파일 생성
	tempDir := os.TempDir()
	// 원본 이미지 URL에서 확장자 추출
	ext := filepath.Ext(strings.Split(imageURL, "?")[0])
	tempFileName := uuid.New().String() + ext
	tempFilePath := filepath.Join(tempDir, tempFileName)

	// 이미지 파일 저장
	file, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("임시 파일 생성 실패: %v", err)
	}
	defer file.Close()

	// 이미지 데이터 복사
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("이미지 저장 실패: %v", err)
	}

	return tempFilePath, nil
}

// runOCR은 이미지 파일에서 텍스트를 추출합니다
func (o *OCRImpl) runOCR(imagePath string) (string, error) {
	// OCR 디버깅용
	fmt.Printf("OCR 실행 - 이미지 경로: %s\n", imagePath)

	// 기본 시도
	cmd := exec.Command("tesseract",
		imagePath,
		"stdout",
		"-l", "kor",
		"--psm", "6",
		"--oem", "3",
		"-c", "preserve_interword_spaces=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("OCR 실행 실패: %v\n", err)
	}

	// OCR 결과 정리
	textDetected := strings.TrimSpace(string(output))

	// 결과가 없으면 다른 psm 모드 시도
	if textDetected == "" || strings.Contains(textDetected, "Estimating") {
		fmt.Printf("OCR 결과 여전히 미흡, 다른 PSM 모드 시도\n")

		psm_modes := []string{"7", "8", "10", "11", "12"}

		for _, psm := range psm_modes {
			cmdAlt := exec.Command("tesseract",
				imagePath,
				"stdout",
				"-l", "kor",
				"--psm", psm)

			outputAlt, _ := cmdAlt.CombinedOutput()
			altText := strings.TrimSpace(string(outputAlt))

			if altText != "" && !strings.Contains(altText, "Estimating") {
				textDetected = altText
				fmt.Printf("PSM %s에서 텍스트 감지됨\n", psm)
				break
			}
		}
	}

	// 디버그 메시지만 있고 실제 텍스트가 없는 경우
	if textDetected == "" || (strings.Contains(textDetected, "Estimating") && len(textDetected) < 100) {
		// 빈 문자열 대신 기본값 반환
		return "[OCR 인식 불가: 이미지에서 텍스트 추출 실패]", nil
	}

	return textDetected, nil
}
