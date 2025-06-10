package detector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	// 시작 시간 기록 (메트릭용)
	startTime := time.Now()
	serviceType := "image_ocr"

	if imageURL == "" {
		utils.RecordError(serviceType, "empty_url")
		return "", fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	// GIF 파일인지 빠르게 확인 (URL 기반)
	if strings.Contains(strings.ToLower(imageURL), ".gif") {
		return "[GIF 파일은 OCR 미지원]", nil
	}

	// 캐시 확인
	// if o.ocrRepo != nil {
	// 	cache, err := o.ocrRepo.GetOCRCache(imageURL)
	// 	if err == nil && cache != nil && cache.TextDetected != "" {
	// 		// 캐시 히트 메트릭 기록
	// 		utils.Info(serviceType, "캐시 히트: %s", imageURL)
	// 		fmt.Printf("cache.TextDetected: %s\n", cache.TextDetected)
	// 		return cache.TextDetected, nil
	// 	}
	// }

	// 상위 컨텍스트 생성 (최대 처리 시간 제한)
	parentCtx, parentCancel := context.WithTimeout(context.Background(), constants.TIMEOUT)
	defer parentCancel()

	// 비동기 처리를 위한 채널
	resultCh := make(chan struct {
		text string
		err  error
	})

	// 비동기로 이미지 다운로드 및 OCR 처리
	go func() {
		// 이미지 다운로드
		utils.Info(serviceType, "이미지 다운로드 시작: %s", imageURL)
		tempFile, err := o.downloadImage(imageURL)
		if err != nil {
			utils.Error(serviceType, "이미지 다운로드 실패 [URL: %s]: %v", imageURL, err)
			utils.RecordError(serviceType, "download_failed")
			resultCh <- struct {
				text string
				err  error
			}{"", err}
			return
		}
		defer os.Remove(tempFile) // 임시 파일 정리

		// 상위 컨텍스트가 취소되었는지 확인
		select {
		case <-parentCtx.Done():
			// 이미 상위 컨텍스트가 취소된 경우
			return
		default:
			// 계속 진행
		}

		// OCR 실행 (상위 컨텍스트 전달)
		textDetected, err := o.runOCR(parentCtx, tempFile, imageURL)

		// 상위 컨텍스트가 취소되었는지 다시 확인
		select {
		case <-parentCtx.Done():
			// 이미 상위 컨텍스트가 취소된 경우
			return
		default:
			// 결과 채널에 전송
			resultCh <- struct {
				text string
				err  error
			}{textDetected, err}
		}
	}()

	// 타임아웃 또는 결과 대기
	select {
	case <-parentCtx.Done():
		// 타임아웃 발생
		timeoutErr := fmt.Sprintf("[OCR 처리 시간 초과 (%s)]", constants.TIMEOUT)
		utils.RecordError("image_ocr_timeout", "OCR 처리 시간 초과")
		return timeoutErr, nil
	case result := <-resultCh:
		// 총 처리 시간 기록 (메트릭용)
		duration := time.Since(startTime).Seconds()
		// 메트릭 기록 - 별도 함수를 사용하여 확실히 실행되도록
		utils.RecordOcrProcessingTime(duration)
		utils.Info(serviceType, "OCR 처리 완료 - 소요 시간: %.2f초", duration)

		// 결과 반환
		if result.err != nil {
			errorMsg := fmt.Sprintf("OCR 처리 실패: %v", result.err)
			utils.Error(serviceType, "처리 실패 [URL: %s]: %s", imageURL, errorMsg)
			utils.RecordError(serviceType, "processing_failed")

			// OCR 오류 로그 데이터 저장
			utils.OCRErrorLog("PROCESSING_FAILED", imageURL, result.err.Error())

			return "[" + errorMsg + "]", nil
		}

		// OCR 결과 캐싱
		if o.ocrRepo != nil && result.text != "" {
			// 비동기 저장 (결과에 영향 없음)
			go func() {
				_ = o.ocrRepo.SaveOCRCache(imageURL, result.text, "image")
			}()
		}

		return result.text, nil
	}
}

// downloadImage는 이미지 URL에서 이미지를 다운로드합니다
func (o *OCRImpl) downloadImage(imageURL string) (string, error) {
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		imageURL = "https://" + imageURL
	}

	// GIF 파일 URL 확인 (경로나 쿼리 파라미터에 .gif가 포함되어 있는지)
	if strings.Contains(strings.ToLower(imageURL), ".gif") {
		return "", fmt.Errorf("GIF 파일은 OCR 미지원: %s", imageURL)
	}

	if !strings.Contains(imageURL, "?type=") && !strings.Contains(imageURL, "&type=") {
		if strings.Contains(imageURL, "?") {
			imageURL += "&type=w773"
		} else {
			imageURL += "?type=w773"
		}
	}

	// 내부 함수: 실제 요청 실행
	doRequest := func(url string, timeout time.Duration) (*http.Response, error) {
		// 타임아웃 컨텍스트 생성
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		// 응답 객체와 에러를 반환할 변수 선언
		var resp *http.Response
		var err error

		// 요청 준비 - HEAD 메서드로 먼저 크기 확인
		headReq, reqErr := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if reqErr == nil {
			headReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

			// HEAD 요청으로 이미지 크기 확인
			client := &http.Client{Timeout: 2 * timeout}
			headResp, headErr := client.Do(headReq)

			if headErr == nil && headResp.StatusCode == http.StatusOK {
				defer headResp.Body.Close()

				// Content-Length 헤더로 이미지 크기 확인
				contentLength := headResp.Header.Get("Content-Length")
				if contentLength != "" {
					if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
						// 이미지 크기 제한 (3MB)
						const maxSize = 3 * 1024 * 1024
						if size > maxSize {
							cancel()
							return nil, fmt.Errorf("이미지 크기가 너무 큼: %.2f MB (최대 %.2f MB)", float64(size)/1024/1024, float64(maxSize)/1024/1024)
						}
					}
				}
			}
		}

		// 요청 준비 - GET 요청으로 실제 이미지 다운로드
		req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
		if reqErr != nil {
			cancel() // 컨텍스트 취소
			return nil, reqErr
		}

		req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Add("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
		req.Header.Add("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

		// HTTP 클라이언트 생성
		client := &http.Client{
			Timeout: (timeout + 2) * timeout,
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

		// Content-Length 헤더 확인 (HEAD 요청을 건너뛴 경우)
		contentLength := resp.Header.Get("Content-Length")
		if contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				// 이미지 크기 제한 (3MB)
				const maxSize = 3 * 1024 * 1024
				if size > maxSize {
					resp.Body.Close()
					cancel()
					return nil, fmt.Errorf("이미지 크기가 너무 큼: %.2f MB (최대 %.2f MB)", float64(size)/1024/1024, float64(maxSize)/1024/1024)
				}
			}
		}

		return resp, nil
	}

	// 1차 시도: 원본 주소 (3초 제한)
	resp, err := doRequest(imageURL, constants.TIMEOUT)

	if err != nil {
		// 이미지 크기 관련 오류인 경우 바로 반환
		if strings.Contains(err.Error(), "이미지 크기가 너무 큼") {
			return "", err
		}

		// 그 외 오류는 프록시로 재시도
		workerURL := configs.GetConfig().Server.WorkerURL + "?url=" + url.QueryEscape(imageURL)
		resp, err = doRequest(workerURL, constants.TIMEOUT)
		if err != nil {
			return "", fmt.Errorf("이미지 요청 실패 (우회 포함): %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("프록시 HTTP 오류 (%d)", resp.StatusCode)
		}
	}

	// 이미지 파일 크기 제한
	// 다운로드 후에도 크기를 확인하여 제한 (Content-Length가 없는 경우 대비)
	tempFilePath, err := utils.SaveResponseToFile(resp, imageURL)
	if err != nil {
		return "", err
	}

	// 파일 크기 확인
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		// 이미지 크기 제한 (3MB)
		const maxSize = 3 * 1024 * 1024
		if fileInfo.Size() > maxSize {
			os.Remove(tempFilePath) // 임시 파일 삭제
			return "", fmt.Errorf("이미지 크기가 너무 큼: %.2f MB (최대 %.2f MB)", float64(fileInfo.Size())/1024/1024, float64(maxSize)/1024/1024)
		}

		// Content-Type 확인 (다운로드 후)
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(strings.ToLower(contentType), "gif") {
			os.Remove(tempFilePath) // 임시 파일 삭제
			return "", fmt.Errorf("GIF 파일은 OCR 미지원: %s (Content-Type: %s)", imageURL, contentType)
		}
	}

	// 이미지 차원(가로/세로) 확인
	dimensions, err := utils.GetImageDimensions(tempFilePath)
	if err == nil {
		// 최적 크롭 적용: 이미지 비율에 따라 가장 좋은 방식으로 잘라냄
		fmt.Printf("이미지 크기 확인: %dx%d\n", dimensions.Width, dimensions.Height)

		// 최적의 OCR 결과를 위해 이미지 크롭 적용
		croppedPath, cropErr := utils.CropImageOptimal(tempFilePath)
		if cropErr != nil {
			fmt.Printf("이미지 최적화 실패: %v. 원본 이미지로 계속 진행합니다.\n", cropErr)
		} else if croppedPath != tempFilePath {
			// 크롭된 이미지가 원본과 다르면 원본 제거하고 크롭된 이미지 사용
			os.Remove(tempFilePath)
			tempFilePath = croppedPath

			// 새 이미지 크기 확인
			if newDimensions, err := utils.GetImageDimensions(tempFilePath); err == nil {
				fmt.Printf("최적화된 이미지 크기: %dx%d\n", newDimensions.Width, newDimensions.Height)
			}
		}
	} else {
		fmt.Printf("이미지 차원 확인 실패 (계속 진행): %v\n", err)
	}

	return tempFilePath, nil
}

// runOCR은 Tesseract를 사용하여 OCR 처리를 수행합니다
func (o *OCRImpl) runOCR(ctx context.Context, imagePath string, imageURL string) (string, error) {
	// OCR 디버깅용
	fmt.Printf("OCR 실행 시작 - 이미지 경로: %s\n", imagePath)
	startTime := time.Now()
	serviceType := "tesseract_ocr"

	// 이미지 형식 확인 - 파일 시그니처 체크
	if utils.IsGifImage(imagePath) {
		fmt.Printf("GIF 이미지 감지됨: %s - OCR 처리 건너뜀\n", imagePath)
		return "[GIF 파일은 OCR 미지원]", nil
	}

	// OCR 처리를 위한 타임아웃 컨텍스트
	ocrCtx, cancel := context.WithTimeout(ctx, constants.TIMEOUT)
	defer cancel()

	// 향상된 OCR 처리 사용 (여러 PSM 모드와 이미지 전처리 적용)
	textDetected := utils.RunTesseractWithContext(ocrCtx, imagePath)

	// 컨텍스트 취소 확인 (상위 컨텍스트의 취소 여부만 확인)
	if ctx.Err() != nil {
		fmt.Printf("상위 컨텍스트 취소됨: %v\n", ctx.Err())
		utils.RecordError(serviceType, "context_deadline_exceeded")
		utils.Error(serviceType, "Tesseract OCR 처리 중 컨텍스트 취소됨: %v [이미지: %s]", ctx.Err(), imageURL)

		// OCR 오류 로그 데이터 저장
		utils.OCRErrorLog("CONTEXT_DEADLINE_EXCEEDED", imageURL, ctx.Err().Error())

		return "context deadline exceeded", nil
	}

	// 텍스트가 없는 경우
	if textDetected == "" {
		fmt.Printf("최종 OCR 결과: 인식 불가 (총 실행 시간: %v)\n", time.Since(startTime))
		utils.RecordError(serviceType, "no_text_detected")
		utils.Error(serviceType, "OCR 인식 불가: 텍스트 추출 실패 [이미지: %s]", imageURL)

		// OCR 오류 로그 데이터 저장
		utils.OCRErrorLog("NO_TEXT_DETECTED", imageURL, "텍스트 추출 실패")

		return "[OCR 인식 불가: 이미지에서 텍스트 추출 실패]", nil
	}

	// OCR 처리 시간 측정 완료
	ocrDuration := time.Since(startTime).Seconds()
	utils.RecordOcrProcessingTime(ocrDuration)

	fmt.Printf("최종 OCR 결과: 성공 (총 실행 시간: %v)\n", time.Since(startTime))

	// 결과 정리 작업은 별도의 goroutine으로 처리하여 빠르게 반환
	resultCh := make(chan string, 1)

	go func() {
		// 한글이 시작되는 부분부터 추출하지 않고 모든 텍스트 유지 (협찬 태그에 영문/특수문자가 포함될 수 있음)
		// 다만 앞부분에 Tesseract 경고 메시지 등이 있다면 제거
		if strings.Contains(textDetected, "Warning:") {
			parts := strings.SplitN(textDetected, "Warning:", 2)
			if len(parts) > 1 {
				// Warning 이후 부분에서 다음 줄바꿈 이후의 텍스트만 사용
				warningParts := strings.SplitN(parts[1], "\n", 2)
				if len(warningParts) > 1 {
					textDetected = warningParts[1]
				}
			}
		}

		// 줄바꿈 제거 및 공백 정리
		textDetected = strings.ReplaceAll(textDetected, "\n", " ")
		textDetected = strings.Join(strings.Fields(textDetected), " ") // 연속된 공백 하나로
		textDetected = strings.TrimSpace(textDetected)

		// 로깅
		fmt.Printf("변환된 OCR 텍스트: %s\n", textDetected)

		// 결과가 너무 길면 잘라서 반환
		if len(textDetected) > 1000 {
			resultCh <- textDetected[:1000]
		} else {
			resultCh <- textDetected
		}
	}()

	// 최대 50ms 내에 정리 작업 완료되지 않으면 원본 결과 반환
	select {
	case result := <-resultCh:
		return result, nil
	case <-time.After(50 * time.Millisecond):
		fmt.Println("텍스트 정리 시간 초과: 원본 텍스트 반환")
		if len(textDetected) > 1000 {
			return textDetected[:1000], nil
		}
		return textDetected, nil
	}
}
