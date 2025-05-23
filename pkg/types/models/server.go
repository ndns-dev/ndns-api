package model

type Server struct {
	// =================== 기본 식별 정보 ===================
	AppName string `json:"app_name" dynamodbav:"AppName"` // 서버 이름 (예: ec2, mac, windows) - 기본 키(Primary Key)
	AppURL  string `json:"app_url" dynamodbav:"AppURL"`   // 서버의 URL
	Type    string `json:"type" dynamodbav:"Type"`        // 서버 유형 (예: api, router, monitoring)
	Region  string `json:"region" dynamodbav:"Region"`    // 서버 지역 (예: ap-northeast-2, home)
	IsCloud bool   `json:"is_cloud" dynamodbav:"IsCloud"` // 클라우드 서버 여부
}
