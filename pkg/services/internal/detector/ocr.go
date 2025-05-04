package detector

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sh5080/ndns-go/pkg/configs"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
)

// OCRService는 이미지에서 텍스트를 추출하는 인터페이스입니다
type OCRService interface {
	// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
	ExtractTextFromImage(imageURL string) (string, error)
}

// OCRImpl는 OCR 서비스 구현체입니다
type OCRImpl struct {
	client  *http.Client
	config  *configs.EnvConfig
	ocrRepo repository.OCRRepository
}

// NewOCRService는 새 OCR 서비스를 생성합니다
func NewOCRService() OCRService {
	return &OCRImpl{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		config:  configs.GetConfig(),
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

	// Tesseract 설치 확인
	if !o.isTesseractInstalled() {
		return "", fmt.Errorf("Tesseract OCR이 설치되지 않았습니다")
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
	resp, err := o.client.Do(req)
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
	tempFileName := uuid.New().String() + ".jpg"
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
	// Tesseract 명령 실행
	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "kor+eng")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Tesseract 실행 실패: %v", err)
	}

	// OCR 결과 정리
	textDetected := strings.TrimSpace(string(output))
	return textDetected, nil
}

// isTesseractInstalled는 Tesseract OCR이 설치되어 있는지 확인합니다
func (o *OCRImpl) isTesseractInstalled() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}
