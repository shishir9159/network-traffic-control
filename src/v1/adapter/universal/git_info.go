package universal

type GitInfo struct {
	GitProvider string `json:"gitProvider"`
	ApiUrl      string `json:"apiUrl"`
	Username    string `json:"Username"`
	SecretKey   string `json:"secretKey"`
}
