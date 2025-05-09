package detector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	constants "github.com/sh5080/ndns-go/pkg/types"
	"github.com/sh5080/ndns-go/pkg/utils"
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
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageURL = "https://" + imageURL
	}

	isNaverImage := false
	for _, pattern := range constants.NAVER_IMAGE_PATTERNS {
		if strings.Contains(imageURL, pattern) {
			isNaverImage = true
			break
		}
	}

	if isNaverImage {
		if !strings.Contains(imageURL, "?type=") && !strings.Contains(imageURL, "&type=") {
			if strings.Contains(imageURL, "?") {
				imageURL += "&type=w773"
			} else {
				imageURL += "?type=w773"
			}
		}
	}

	// 내부 함수: 실제 요청 실행
	doRequest := func(url string, timeout time.Duration) (*http.Response, error) {
		// 타임아웃 컨텍스트 생성 (이미지 전체를 다운로드하는 데 필요한 충분한 시간 제공)
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)

		// 응답 객체와 에러를 반환할 변수 선언
		var resp *http.Response
		var err error
		// 요청 준비
		req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
		if reqErr != nil {
			cancel() // 컨텍스트 취소
			return nil, reqErr
		}

		req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Add("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
		req.Header.Add("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

		// HTTP 클라이언트 생성 - 컨텍스트 타임아웃과는 별도로 클라이언트 타임아웃도 설정
		client := &http.Client{
			Timeout: (timeout + 2) * time.Second, // 컨텍스트보다 약간 더 긴 타임아웃
		}

		resp, err = client.Do(req)

		// 오류가 발생하면 컨텍스트를 취소하고 결과 반환
		if err != nil {
			cancel()
			return nil, err
		}

		// 성공적인 응답이 아니면 컨텍스트 취소
		if resp.StatusCode != http.StatusOK {
			cancel()
			return resp, nil
		}

		// 응답이 성공적이면 컨텍스트 취소는 defer로 미룸
		// 이미지 다운로드가 완료된 후에 취소됨
		return resp, nil
	}

	// 1차 시도: 원본 주소 (3초 제한)
	resp, err := doRequest(imageURL, 3)

	if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
		// 2차 시도: Cloudflare Worker 프록시 경유
		workerURL := configs.GetConfig().Server.WorkerURL + "?url=" + url.QueryEscape(imageURL)
		resp, err = doRequest(workerURL, 3)
		if err != nil {
			return "", fmt.Errorf("이미지 요청 실패 (우회 포함): %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("프록시 HTTP 오류 (%d)", resp.StatusCode)
		}
	}
	return utils.SaveResponseToFile(resp, imageURL)
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
