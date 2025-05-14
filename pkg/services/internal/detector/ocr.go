package detector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	if imageURL == "" {
		return "", fmt.Errorf("이미지 URL이 비어 있습니다")
	}

	// GIF 파일인지 빠르게 확인 (URL 기반)
	if strings.Contains(strings.ToLower(imageURL), ".gif") {
		return "[GIF 파일은 OCR 미지원]", nil
	}

	// 캐시 확인
	if o.ocrRepo != nil {
		cache, err := o.ocrRepo.GetOCRCache(imageURL)
		if err == nil && cache != nil && cache.TextDetected != "" {
			return cache.TextDetected, nil
		}
	}

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
		// 이미지 다운로드 (상위 컨텍스트 전달)
		tempFile, err := o.downloadImage(imageURL)
		if err != nil {
			select {
			case <-parentCtx.Done():
				// 이미 상위 컨텍스트가 취소된 경우 응답 전송 생략
				return
			default:
				resultCh <- struct {
					text string
					err  error
				}{"", fmt.Errorf("이미지 다운로드 실패: %v", err)}
			}
			return
		}

		// tempFile이 생성되었으면 처리 완료 후 삭제
		defer func() {
			if tempFile != "" {
				os.Remove(tempFile) // 임시 파일 정리
			}
		}()

		// 상위 컨텍스트가 취소되었는지 확인
		select {
		case <-parentCtx.Done():
			// 이미 상위 컨텍스트가 취소된 경우
			return
		default:
			// 계속 진행
		}

		// OCR 실행 (상위 컨텍스트 전달)
		textDetected, err := o.runOCR(parentCtx, tempFile)

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
		timeoutErr := fmt.Sprintf("OCR 처리 시간 초과 (%s): %v", constants.TIMEOUT, parentCtx.Err())
		fmt.Printf("상위 컨텍스트 타임아웃: %s\n", timeoutErr)
		return "[" + timeoutErr + "]", nil
	case result := <-resultCh:
		// 결과 반환
		if result.err != nil {
			errorMsg := fmt.Sprintf("OCR 처리 실패: %v", result.err)
			fmt.Printf("%s\n", errorMsg)
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
		// 타임아웃 컨텍스트 생성
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)

		// 응답 객체와 에러를 반환할 변수 선언
		var resp *http.Response
		var err error

		// 요청 준비 - HEAD 메서드로 먼저 크기 확인
		headReq, reqErr := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if reqErr == nil {
			headReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

			// HEAD 요청으로 이미지 크기 확인
			client := &http.Client{Timeout: 2 * time.Second}
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
			Timeout: (timeout + 2) * time.Second,
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
		// 가로 또는 세로 크기가 최대값을 초과하는 경우
		const maxDimension = constants.MAX_IMAGE_DIMENSION // 픽셀 단위 (4000x4000 이상인 이미지는 처리하지 않음)
		if dimensions.Width > maxDimension || dimensions.Height > maxDimension {
			// 큰 이미지의 경우 상단 1000픽셀만 자르기
			const cropHeight = constants.CROP_HEIGHT // 상단 1000픽셀만 사용
			fmt.Printf("이미지가 너무 큼: %dx%d. 상단 %d픽셀만 사용합니다.\n",
				dimensions.Width, dimensions.Height, cropHeight)

			croppedPath, cropErr := utils.CropImageTop(tempFilePath, cropHeight)
			if cropErr != nil {
				fmt.Printf("이미지 자르기 실패: %v. 원본 이미지로 계속 진행합니다.\n", cropErr)
			} else {
				// 원본 파일은 더 이상 필요하지 않음
				os.Remove(tempFilePath)
				// 잘린 이미지 경로로 업데이트
				tempFilePath = croppedPath

				// 새 이미지 크기 확인
				if newDimensions, err := utils.GetImageDimensions(tempFilePath); err == nil {
					fmt.Printf("잘린 이미지 크기: %dx%d\n", newDimensions.Width, newDimensions.Height)
				}
			}
		}

		// 이미지 총 픽셀 수가 너무 많은 경우 (초대형 이미지)
		const maxPixels = constants.MAX_IMAGE_SIZE // 1200만 픽셀 (약 4000x3000 크기)
		totalPixels := dimensions.Width * dimensions.Height
		if totalPixels > maxPixels {
			// 이미 이미지를 잘랐으면 추가적인 조치 필요 없음
			if !strings.Contains(tempFilePath, "_cropped.jpg") {
				fmt.Printf("이미지 총 픽셀 수가 너무 많음: %d (최대 %d). 이미지를 자릅니다.\n",
					totalPixels, maxPixels)

				const cropHeight = constants.CROP_HEIGHT // 상단 1000픽셀만 사용
				croppedPath, cropErr := utils.CropImageTop(tempFilePath, cropHeight)
				if cropErr != nil {
					fmt.Printf("이미지 자르기 실패: %v. 원본 이미지로 계속 진행합니다.\n", cropErr)
				} else {
					// 원본 파일은 더 이상 필요하지 않음
					os.Remove(tempFilePath)
					// 잘린 이미지 경로로 업데이트
					tempFilePath = croppedPath

					// 새 이미지 크기 확인
					if newDimensions, err := utils.GetImageDimensions(tempFilePath); err == nil {
						fmt.Printf("잘린 이미지 크기: %dx%d\n", newDimensions.Width, newDimensions.Height)
					}
				}
			}
		}
	} else {
		fmt.Printf("이미지 차원 확인 실패 (계속 진행): %v\n", err)
	}

	return tempFilePath, nil
}

// runOCR은 이미지 파일에서 텍스트를 추출합니다
func (o *OCRImpl) runOCR(ctx context.Context, imagePath string) (string, error) {
	// OCR 디버깅용
	fmt.Printf("OCR 실행 시작 - 이미지 경로: %s\n", imagePath)
	startTime := time.Now()

	// 이미지 형식 확인 - 파일 시그니처 체크
	if utils.IsGifImage(imagePath) {
		fmt.Printf("GIF 이미지 감지됨: %s - OCR 처리 건너뜀\n", imagePath)
		return "[GIF 파일은 OCR 미지원]", nil
	}

	// OCR 처리를 위한 타임아웃 컨텍스트
	ocrCtx, cancel := context.WithTimeout(ctx, constants.TIMEOUT)
	defer cancel()

	// 명령 실행을 위한 채널
	doneCh := make(chan struct {
		output []byte
		err    error
	}, 1)

	// 기본 시도
	cmd := exec.CommandContext(ocrCtx,
		"tesseract",
		imagePath,
		"stdout",
		"-l", "kor",
		"--psm", "6",
		"--oem", "3",
		"-c", "preserve_interword_spaces=1")

	fmt.Printf("OCR 명령 실행: %s\n", cmd.String())

	// 비동기로 명령 실행
	go func() {
		output, err := cmd.CombinedOutput()
		doneCh <- struct {
			output []byte
			err    error
		}{output, err}
	}()

	// 타임아웃 또는 결과 대기
	var output []byte
	var err error
	select {
	case <-ctx.Done():
		cancel() // 확실히 취소
		fmt.Printf("상위 컨텍스트 취소됨: %v\n", ctx.Err())
		return "[OCR 타임아웃: 상위 컨텍스트 취소]", nil
	case <-ocrCtx.Done():
		// kill 프로세스
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		fmt.Printf("OCR 컨텍스트 취소됨: %v\n", ocrCtx.Err())
		return "[OCR 타임아웃: 처리 시간 초과]", nil
	case result := <-doneCh:
		output, err = result.output, result.err
	}

	execTime := time.Since(startTime)
	if err != nil {
		fmt.Printf("OCR 실행 실패 (소요시간: %v): %v\n", execTime, err)
	} else {
		fmt.Printf("OCR 실행 완료 (소요시간: %v)\n", execTime)
	}

	// 컨텍스트 상태 다시 확인
	if ctx.Err() != nil {
		fmt.Printf("OCR 처리 후 상위 컨텍스트 취소 확인: %v\n", ctx.Err())
		return "[OCR 타임아웃: 처리 시간 초과]", nil
	}

	// OCR 결과 정리
	textDetected := strings.TrimSpace(string(output))
	textLength := len(textDetected)

	// 결과 요약 (디버깅용)
	if textLength > 100 {
		fmt.Printf("OCR 결과: %s... (총 %d자)\n", textDetected[:100], textLength)
	} else {
		fmt.Printf("OCR 결과: %s (총 %d자)\n", textDetected, textLength)
	}

	// 결과가 없으면 다른 psm 모드 시도
	if textDetected == "" || strings.Contains(textDetected, "Estimating") {
		fmt.Printf("OCR 결과 여전히 미흡, 다른 PSM 모드 시도\n")

		psm_modes := []string{"7", "8", "10", "11", "12"}

		for _, psm := range psm_modes {
			// 컨텍스트가 취소되었는지 확인
			if ctx.Err() != nil {
				fmt.Printf("대체 OCR 시도 전 컨텍스트 취소 확인: %v\n", ctx.Err())
				return "[OCR 타임아웃: 처리 시간 초과]", nil
			}

			// 각 PSM 모드별 타임아웃 컨텍스트
			psmCtx, psmCancel := context.WithTimeout(ctx, constants.TIMEOUT)

			// 명령 실행을 위한 채널
			psmDoneCh := make(chan struct {
				output []byte
				err    error
			}, 1)

			altStartTime := time.Now()
			cmdAlt := exec.CommandContext(psmCtx,
				"tesseract",
				imagePath,
				"stdout",
				"-l", "kor",
				"--psm", psm)

			fmt.Printf("대체 OCR 명령 실행 (PSM %s): %s\n", psm, cmdAlt.String())

			// 비동기로 명령 실행
			go func() {
				output, err := cmdAlt.CombinedOutput()
				psmDoneCh <- struct {
					output []byte
					err    error
				}{output, err}
			}()

			// 타임아웃 또는 결과 대기
			var outputAlt []byte
			var cmdErr error
			select {
			case <-ctx.Done():
				psmCancel()
				if cmdAlt.Process != nil {
					_ = cmdAlt.Process.Kill()
				}
				fmt.Printf("대체 OCR(PSM %s) 상위 컨텍스트 취소: %v\n", psm, ctx.Err())
				continue
			case <-psmCtx.Done():
				if cmdAlt.Process != nil {
					_ = cmdAlt.Process.Kill()
				}
				fmt.Printf("대체 OCR(PSM %s) 타임아웃: %v\n", psm, psmCtx.Err())
				psmCancel()
				continue
			case result := <-psmDoneCh:
				outputAlt, cmdErr = result.output, result.err
			}

			psmCancel() // 컨텍스트 취소
			altExecTime := time.Since(altStartTime)

			if cmdErr != nil {
				fmt.Printf("대체 OCR(PSM %s) 실행 실패 (소요시간: %v): %v\n", psm, altExecTime, cmdErr)
			} else {
				fmt.Printf("대체 OCR(PSM %s) 실행 완료 (소요시간: %v)\n", psm, altExecTime)
			}

			altText := strings.TrimSpace(string(outputAlt))
			altTextLength := len(altText)

			if altText != "" && !strings.Contains(altText, "Estimating") {
				textDetected = altText
				fmt.Printf("PSM %s에서 텍스트 감지됨 (총 %d자)\n", psm, altTextLength)
				break
			}
		}
	}

	// 최종 컨텍스트 확인
	if ctx.Err() != nil {
		fmt.Printf("OCR 완료 후 최종 컨텍스트 취소 확인: %v\n", ctx.Err())
		return "[OCR 타임아웃: 처리 시간 초과]", nil
	}

	// 디버그 메시지만 있고 실제 텍스트가 없는 경우
	if textDetected == "" || (strings.Contains(textDetected, "Estimating") && len(textDetected) < 100) {
		// 빈 문자열 대신 기본값 반환
		fmt.Printf("최종 OCR 결과: 인식 불가 (총 실행 시간: %v)\n", time.Since(startTime))
		return "[OCR 인식 불가: 이미지에서 텍스트 추출 실패]", nil
	}

	fmt.Printf("최종 OCR 결과: 성공 (총 실행 시간: %v)\n", time.Since(startTime))
	return textDetected[:min(100, len(textDetected))], nil
}
