package structure

// NgrokTunnel은 개별 ngrok 터널의 정보를 나타냅니다.
type NgrokTunnel struct {
	Name      string `json:"name"`
	ID        string `json:"ID"`
	URI       string `json:"uri"`
	PublicURL string `json:"public_url"`
	Proto     string `json:"proto"`
	Config    struct {
		Addr string `json:"addr"`
	} `json:"config"`
}

// NgrokTunnelsResponse는 ngrok API의 터널 목록 응답을 나타냅니다.
type NgrokTunnelsResponse struct {
	Tunnels []NgrokTunnel `json:"tunnels"`
	URI     string        `json:"uri"`
}
