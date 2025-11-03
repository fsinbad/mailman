package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type UpdateOAuth2TokenRequest struct {
	EmailAddress    string                 `json:"email_address"`
	CustomSettings  map[string]interface{} `json:"custom_settings"`
}

func main() {
	fmt.Println("=== OAuth2 Token Update Test Tool ===")

	// 测试配置 - 请根据实际情况修改
	baseURL := "http://localhost:8080"
	token := "13ed62998db168c76af1ac4ba667737ede099df7c073c0c4c1afc303fc37d3a3"
	email := "mwasmkale@gmail.com"

	// 测试OAuth2 token更新
	fmt.Printf("Testing OAuth2 token update for: %s\n", email)

	// 准备测试数据
	testData := UpdateOAuth2TokenRequest{
		EmailAddress: email,
		CustomSettings: map[string]interface{}{
			"client_id":     "your_google_client_id.apps.googleusercontent.com",
			"access_token":  "your_access_token_here",
			"refresh_token": "your_refresh_token_here",
			"expires_at":    fmt.Sprintf("%d", time.Now().Unix()+3600),
			"token_type":    "Bearer",
		},
	}

	// 转换为JSON
	jsonData, err := json.Marshal(testData)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/api/accounts/update-oauth2-token", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	// 显示结果
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ OAuth2 token update successful!")

		// 解析响应以显示更新的账户信息
		var account map[string]interface{}
		if err := json.Unmarshal(body, &account); err == nil {
			fmt.Printf("Updated Account ID: %.0f\n", account["id"])
			fmt.Printf("Email: %s\n", account["emailAddress"])
			fmt.Printf("Error Status: %v\n", account["errorStatus"])
			fmt.Printf("Error Count: %v\n", account["errorCount"])

			if customSettings, ok := account["customSettings"].(map[string]interface{}); ok {
				fmt.Printf("Access Token Length: %d\n", len(fmt.Sprintf("%v", customSettings["access_token"])))
				fmt.Printf("Refresh Token Length: %d\n", len(fmt.Sprintf("%v", customSettings["refresh_token"])))
			}
		}
	} else {
		fmt.Println("❌ OAuth2 token update failed!")
	}
}