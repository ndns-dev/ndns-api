package structures

import "time"

// OCRCache는 OCR 결과를 캐싱하기 위한 구조체입니다
type OCRCache struct {
	ImageURL     string    `json:"imageURL"`
	TextDetected string    `json:"textDetected"`
	ImageType    string    `json:"imageType"`
	DetectedAt   time.Time `json:"detectedAt"`
}
