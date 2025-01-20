package utility

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type IPInfoResponse struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
}

func GetTimeZoneFromIP() (string, error) {
	url := fmt.Sprintf("https://ipinfo.io/json")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("неожиданный статус ответа: %d", resp.StatusCode)
	}

	var ipInfo IPInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return "", fmt.Errorf("ошибка декодирования ответа: %v", err)
	}

	return ipInfo.Timezone, nil
}
