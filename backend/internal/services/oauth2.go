package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mailman/internal/models"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

// OAuth2Config represents OAuth2 configuration
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	TokenURL     string
}

// TokenCacheEntry represents a cached token entry
type TokenCacheEntry struct {
	AccessToken string
	ExpiresAt   time.Time
	RefreshTime time.Time
}

// OAuth2Service handles OAuth2 authentication
type OAuth2Service struct {
	httpClient   *http.Client
	tokenCache   map[string]*TokenCacheEntry
	cacheMutex   sync.RWMutex
	accountLocks map[string]*sync.Mutex // 基于账户ID的锁
	locksMutex   sync.RWMutex           // 保护 accountLocks map 的锁
	db           *gorm.DB               // 数据库连接
}

// NewOAuth2Service creates a new OAuth2Service
func NewOAuth2Service(db *gorm.DB) *OAuth2Service {
	return &OAuth2Service{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenCache:   make(map[string]*TokenCacheEntry),
		accountLocks: make(map[string]*sync.Mutex),
		db:           db,
	}
}

// createHTTPClientWithProxy creates an HTTP client with proxy support
func (s *OAuth2Service) createHTTPClientWithProxy(proxyStr string) (*http.Client, error) {
	if proxyStr == "" {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	// Create transport with proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Handle SOCKS5 proxy
	if proxyURL.Scheme == "socks5" || proxyURL.Scheme == "socks5h" {
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 proxy dialer: %w", err)
		}
		transport.Dial = dialer.Dial
		transport.Proxy = nil // Don't use HTTP proxy for SOCKS5
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

// getAccountLock 获取账户特定的锁，避免不同账户互相阻塞
func (s *OAuth2Service) getAccountLock(accountKey string) *sync.Mutex {
	s.locksMutex.RLock()
	lock, exists := s.accountLocks[accountKey]
	s.locksMutex.RUnlock()

	if exists {
		return lock
	}

	// 如果锁不存在，需要创建新锁
	s.locksMutex.Lock()
	defer s.locksMutex.Unlock()

	// 双重检查，防止并发创建
	if lock, exists := s.accountLocks[accountKey]; exists {
		return lock
	}

	// 创建新锁
	lock = &sync.Mutex{}
	s.accountLocks[accountKey] = lock
	return lock
}

// getCacheKey 生成缓存键
func (s *OAuth2Service) getCacheKey(providerType, clientID, refreshToken string) string {
	// 简化缓存键生成，避免MD5依赖
	return fmt.Sprintf("%s_%s_%s", providerType, clientID, refreshToken[:10])
}

// RefreshAccessTokenWithCache 带缓存和并发控制的token刷新
func (s *OAuth2Service) RefreshAccessTokenWithCache(providerType, clientID, clientSecret, refreshToken string, accountID uint) (string, error) {
	return s.RefreshAccessTokenWithCacheAndProxy(providerType, clientID, clientSecret, refreshToken, accountID, "")
}

// RefreshAccessTokenWithCacheAndProxy 带缓存、并发控制和代理支持的token刷新
func (s *OAuth2Service) RefreshAccessTokenWithCacheAndProxy(providerType, clientID, clientSecret, refreshToken string, accountID uint, proxy string) (string, error) {
	cacheKey := s.getCacheKey(providerType, clientID, refreshToken)
	accountKey := fmt.Sprintf("%s_%d", providerType, accountID)

	// 先检查缓存
	s.cacheMutex.RLock()
	if entry, exists := s.tokenCache[cacheKey]; exists {
		// 检查token是否还有效（提前5分钟过期）
		if time.Now().Before(entry.ExpiresAt.Add(-5 * time.Minute)) {
			s.cacheMutex.RUnlock()
			fmt.Printf("OAuth2: Using cached token for account %d, expires at: %v\n", accountID, entry.ExpiresAt)
			return entry.AccessToken, nil
		}
	}
	s.cacheMutex.RUnlock()

	// 获取账户特定的锁
	accountLock := s.getAccountLock(accountKey)
	accountLock.Lock()
	defer accountLock.Unlock()

	// 锁定后再次检查缓存（双重检查锁定模式）
	s.cacheMutex.RLock()
	if entry, exists := s.tokenCache[cacheKey]; exists {
		if time.Now().Before(entry.ExpiresAt.Add(-5 * time.Minute)) {
			s.cacheMutex.RUnlock()
			fmt.Printf("OAuth2: Using cached token after lock for account %d, expires at: %v\n", accountID, entry.ExpiresAt)
			return entry.AccessToken, nil
		}
	}
	s.cacheMutex.RUnlock()

	// 防止频繁刷新：如果上次刷新时间在30秒内，等待一下
	s.cacheMutex.RLock()
	if entry, exists := s.tokenCache[cacheKey]; exists {
		if time.Since(entry.RefreshTime) < 30*time.Second {
			s.cacheMutex.RUnlock()
			fmt.Printf("OAuth2: Throttling refresh for account %d, last refresh: %v\n", accountID, entry.RefreshTime)
			return "", fmt.Errorf("token refresh throttled, please wait a moment")
		}
	}
	s.cacheMutex.RUnlock()

	fmt.Printf("OAuth2: Refreshing token for account %d (provider: %s) with proxy: %s\n", accountID, providerType, proxy)

	// 刷新token - 使用代理支持的方法
	newAccessToken, newRefreshToken, err := s.RefreshAccessTokenForProviderWithProxy(providerType, clientID, clientSecret, refreshToken, proxy)
	if err != nil {
		return "", err
	}

	// 如果有新的refresh token，更新到数据库
	if newRefreshToken != "" && s.db != nil {
		fmt.Printf("OAuth2: Updating refresh token for account %d (new token length: %d)\n", accountID, len(newRefreshToken))

		// 更新数据库中的refresh token
		result := s.db.Model(&models.EmailAccount{}).
			Where("id = ?", accountID).
			Update("token", newRefreshToken)

		if result.Error != nil {
			fmt.Printf("OAuth2: Failed to update refresh token in database for account %d: %v\n", accountID, result.Error)
		} else {
			fmt.Printf("OAuth2: Successfully updated refresh token in database for account %d\n", accountID)
		}
	}

	// 更新缓存
	s.cacheMutex.Lock()
	s.tokenCache[cacheKey] = &TokenCacheEntry{
		AccessToken: newAccessToken,
		ExpiresAt:   time.Now().Add(55 * time.Minute), // 比实际过期时间早5分钟
		RefreshTime: time.Now(),
	}
	s.cacheMutex.Unlock()

	fmt.Printf("OAuth2: Token refreshed and cached for account %d\n", accountID)
	return newAccessToken, nil
}

// RefreshAccessToken refreshes the access token using refresh token (legacy method for Outlook)
func (s *OAuth2Service) RefreshAccessToken(clientID, refreshToken string) (string, error) {
	// Legacy method - assumes empty client_secret for backward compatibility
	return s.RefreshAccessTokenForProvider("outlook", clientID, "", refreshToken)
}

// RefreshAccessTokenForProvider refreshes the access token for a specific provider
func (s *OAuth2Service) RefreshAccessTokenForProvider(providerType string, clientID, clientSecret, refreshToken string) (string, error) {
	var tokenURL, scope string

	switch providerType {
	case "gmail":
		tokenURL = "https://oauth2.googleapis.com/token"
		scope = "https://mail.google.com/ https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	case "outlook":
		tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
		scope = "offline_access https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/POP.AccessAsUser.All https://outlook.office.com/SMTP.Send"
	default:
		return "", fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Log the request for debugging (hide sensitive data)
	fmt.Printf("OAuth2: Refreshing token for provider: %s, client_id: %s\n", providerType, clientID)
	fmt.Printf("OAuth2: Refresh token length: %d\n", len(refreshToken))

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("scope", scope)

	// Only include client_secret for confidential clients (not public clients)
	// For Microsoft public clients, client_secret should be empty or not included
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
		fmt.Printf("OAuth2: Using confidential client (with client_secret) for %s\n", providerType)
	} else {
		fmt.Printf("OAuth2: Using public client (no client_secret) for %s\n", providerType)
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log response status for debugging
	fmt.Printf("OAuth2: Response status: %d\n", resp.StatusCode)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Log raw response if JSON parsing fails
		fmt.Printf("OAuth2: Failed to parse JSON. Raw response: %s\n", string(body))
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if errorMsg, ok := result["error"]; ok {
		errorDesc, _ := result["error_description"].(string)
		errorCodes, _ := result["error_codes"].([]interface{})
		correlationId, _ := result["correlation_id"].(string)

		// Provide detailed error information
		errInfo := fmt.Sprintf("OAuth2 error: %v - %s", errorMsg, errorDesc)
		if len(errorCodes) > 0 {
			errInfo += fmt.Sprintf(" (Error codes: %v)", errorCodes)
		}
		if correlationId != "" {
			errInfo += fmt.Sprintf(" (Correlation ID: %s)", correlationId)
		}

		// Common error explanations
		if errorMsg == "invalid_grant" {
			errInfo += "\nPossible causes: 1) Refresh token expired 2) Token already used 3) Invalid client_id 4) User revoked permissions"
		}

		return "", fmt.Errorf(errInfo)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		// Log the entire response for debugging
		fmt.Printf("OAuth2: No access_token in response. Full response: %+v\n", result)
		return "", fmt.Errorf("access_token not found in response")
	}

	// 获取过期时间信息
	expiresIn, _ := result["expires_in"].(float64)
	fmt.Printf("OAuth2: Successfully obtained access token (length: %d), expires in: %.0f seconds\n", len(accessToken), expiresIn)

	return accessToken, nil
}

// RefreshAccessTokenForProviderWithProxy refreshes the access token for a specific provider with proxy support
// Returns new access token and new refresh token (if provided by the provider)
func (s *OAuth2Service) RefreshAccessTokenForProviderWithProxy(providerType string, clientID, clientSecret, refreshToken, proxy string) (accessToken string, newRefreshToken string, err error) {
	var tokenURL, scope string

	switch providerType {
	case "gmail":
		tokenURL = "https://oauth2.googleapis.com/token"
		scope = "https://mail.google.com/ https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	case "outlook":
		tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
		scope = "offline_access https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/POP.AccessAsUser.All https://outlook.office.com/SMTP.Send"
	default:
		return "", "", fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Log the request for debugging (hide sensitive data)
	fmt.Printf("OAuth2: Refreshing token for provider: %s, client_id: %s with proxy: %s\n", providerType, clientID, proxy)
	fmt.Printf("OAuth2: Refresh token length: %d\n", len(refreshToken))

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("scope", scope)

	// Only include client_secret for confidential clients (not public clients)
	// For Microsoft public clients, client_secret should be empty or not included
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
		fmt.Printf("OAuth2 Proxy: Using confidential client (with client_secret) for %s\n", providerType)
	} else {
		fmt.Printf("OAuth2 Proxy: Using public client (no client_secret) for %s\n", providerType)
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 创建支持代理的HTTP客户端
	client, err := s.createHTTPClientWithProxy(proxy)
	if err != nil {
		return "", "", fmt.Errorf("failed to create HTTP client with proxy: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log response status for debugging
	fmt.Printf("OAuth2: Response status: %d\n", resp.StatusCode)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Log raw response if JSON parsing fails
		fmt.Printf("OAuth2: Failed to parse JSON. Raw response: %s\n", string(body))
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if errorMsg, ok := result["error"]; ok {
		errorDesc, _ := result["error_description"].(string)
		errorCodes, _ := result["error_codes"].([]interface{})
		correlationId, _ := result["correlation_id"].(string)

		// Provide detailed error information
		errInfo := fmt.Sprintf("OAuth2 error: %v - %s", errorMsg, errorDesc)
		if len(errorCodes) > 0 {
			errInfo += fmt.Sprintf(" (Error codes: %v)", errorCodes)
		}
		if correlationId != "" {
			errInfo += fmt.Sprintf(" (Correlation ID: %s)", correlationId)
		}

		// Common error explanations
		if errorMsg == "invalid_grant" {
			errInfo += "\nPossible causes: 1) Refresh token expired 2) Token already used 3) Invalid client_id 4) User revoked permissions"
		}

		return "", "", fmt.Errorf(errInfo)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		// Log the entire response for debugging
		fmt.Printf("OAuth2: No access_token in response. Full response: %+v\n", result)
		return "", "", fmt.Errorf("access_token not found in response")
	}

	// Check if a new refresh token was provided
	newRefreshToken = ""
	if newRefresh, ok := result["refresh_token"].(string); ok && newRefresh != "" {
		newRefreshToken = newRefresh
		fmt.Printf("OAuth2: New refresh token provided (length: %d)\n", len(newRefreshToken))
	} else {
		fmt.Printf("OAuth2: No new refresh token provided, keeping existing one\n")
	}

	// 获取过期时间信息
	expiresIn, _ := result["expires_in"].(float64)
	fmt.Printf("OAuth2: Successfully obtained access token (length: %d), expires in: %.0f seconds\n", len(accessToken), expiresIn)

	return accessToken, newRefreshToken, nil
}

// GenerateAuthURL generates OAuth2 authorization URL for a provider
func (s *OAuth2Service) GenerateAuthURL(providerType string, clientID, redirectURI, state string) (string, error) {
	var authURL, scope string

	switch providerType {
	case "gmail":
		authURL = "https://accounts.google.com/o/oauth2/auth"
		scope = "https://mail.google.com/ https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	case "outlook":
		authURL = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
		scope = "offline_access https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/POP.AccessAsUser.All https://outlook.office.com/SMTP.Send"
	default:
		return "", fmt.Errorf("unsupported provider type: %s", providerType)
	}

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return fmt.Sprintf("%s?%s", authURL, params.Encode()), nil
}

// ExchangeCodeForTokens exchanges authorization code for tokens
func (s *OAuth2Service) ExchangeCodeForTokens(providerType, clientID, clientSecret, code, redirectURI string) (accessToken, refreshToken string, err error) {
	var tokenURL string

	switch providerType {
	case "gmail":
		tokenURL = "https://oauth2.googleapis.com/token"
	case "outlook":
		tokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	default:
		return "", "", fmt.Errorf("unsupported provider type: %s", providerType)
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if errorMsg, ok := result["error"]; ok {
		errorDesc, _ := result["error_description"].(string)
		return "", "", fmt.Errorf("OAuth2 error: %v - %s", errorMsg, errorDesc)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", "", fmt.Errorf("access_token not found in response")
	}

	refreshToken, ok = result["refresh_token"].(string)
	if !ok {
		return "", "", fmt.Errorf("refresh_token not found in response")
	}

	return accessToken, refreshToken, nil
}

// GenerateOAuth2AuthString generates the OAuth2 authentication string for IMAP
func (s *OAuth2Service) GenerateOAuth2AuthString(email, accessToken string) string {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", email, accessToken)
	return base64.StdEncoding.EncodeToString([]byte(authString))
}

// OAuth2SASLClient implements the SASL XOAUTH2 mechanism
type OAuth2SASLClient struct {
	email       string
	accessToken string
}

// NewOAuth2SASLClient creates a new OAuth2 SASL client
func NewOAuth2SASLClient(email, accessToken string) sasl.Client {
	return &OAuth2SASLClient{
		email:       email,
		accessToken: accessToken,
	}
}

// Start begins the SASL authentication
func (c *OAuth2SASLClient) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte(fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.email, c.accessToken))
	fmt.Printf("OAuth2 SASL: Starting authentication for %s\n", c.email)
	fmt.Printf("OAuth2 SASL: Access token length: %d\n", len(c.accessToken))
	fmt.Printf("OAuth2 SASL: Initial response length: %d\n", len(ir))
	return
}

// Next continues the SASL authentication
func (c *OAuth2SASLClient) Next(challenge []byte) (response []byte, err error) {
	fmt.Printf("OAuth2 SASL: Next called with challenge length: %d\n", len(challenge))
	if len(challenge) > 0 {
		fmt.Printf("OAuth2 SASL: Challenge content: %s\n", string(challenge))
	}
	// OAuth2 doesn't require additional steps
	return nil, nil
}
