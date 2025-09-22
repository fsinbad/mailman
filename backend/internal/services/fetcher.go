package services

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mailman/internal/models"
	"mailman/internal/repository"
	"mailman/internal/utils"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"golang.org/x/net/proxy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

// FetcherService is responsible for fetching emails from an IMAP server.
type FetcherService struct {
	accountRepo   *repository.EmailAccountRepository
	emailRepo     *repository.EmailRepository
	parserService *ParserService
	oauth2Service *OAuth2Service
	logger        *utils.Logger
}

// FetchEmailsOptions contains options for fetching emails
type FetchEmailsOptions struct {
	Mailbox         string
	Limit           int
	Offset          int
	StartDate       *time.Time
	EndDate         *time.Time
	SearchQuery     string
	FetchFromServer bool
	IncludeBody     bool
	SortBy          string
	Folders         []string // List of folders to fetch from
}

// NewFetcherService creates a new FetcherService.
func NewFetcherService(accountRepo *repository.EmailAccountRepository, emailRepo *repository.EmailRepository, db *gorm.DB) *FetcherService {
	return &FetcherService{
		accountRepo:   accountRepo,
		emailRepo:     emailRepo,
		parserService: NewParserService(),
		oauth2Service: NewOAuth2Service(db),
		logger:        utils.NewLogger("FetcherService"),
	}
}

// FetchEmails fetches emails for a given account with default options.
func (s *FetcherService) FetchEmails(account models.EmailAccount) ([]models.Email, error) {
	s.logger.Debug("FetchEmails called for account %s", account.EmailAddress)
	return s.FetchEmailsWithOptions(account, FetchEmailsOptions{
		Mailbox:         "INBOX",
		Limit:           10,
		FetchFromServer: true,
		IncludeBody:     true,
		SortBy:          "date_desc",
	})
}

// FetchEmailsWithOptions fetches emails for a given account with specified options.
func (s *FetcherService) FetchEmailsWithOptions(account models.EmailAccount, options FetchEmailsOptions) ([]models.Email, error) {
	s.logger.Info("FetchEmailsWithOptions called for account %s, mailbox: %s, limit: %d, fetchFromServer: %v",
		account.EmailAddress, options.Mailbox, options.Limit, options.FetchFromServer)

	// If not fetching from server, use database queries
	if !options.FetchFromServer {
		s.logger.Debug("Fetching emails from database for account %d", account.ID)
		return s.fetchEmailsFromDatabase(account.ID, options)
	}

	// Fetch from IMAP server
	s.logger.Debug("Fetching emails from IMAP server for account %s", account.EmailAddress)
	return s.fetchEmailsFromServer(account, options)
}

// FetchEmailsFromMultipleMailboxes fetches emails from multiple mailboxes based on user selection
func (s *FetcherService) FetchEmailsFromMultipleMailboxes(account models.EmailAccount, options FetchEmailsOptions) ([]models.Email, error) {
	s.logger.Debug("Starting to fetch emails from multiple mailboxes for %s", account.EmailAddress)
	if options.StartDate != nil {
		s.logger.Debug("Filter StartDate: %s", options.StartDate.Format(time.RFC3339))
	}

	// 检查是否是Gmail账户
	isGmailAccount := account.AuthType == "oauth2" && (account.EmailAddress == "" ||
		strings.Contains(account.EmailAddress, "@gmail.com") ||
		strings.Contains(account.EmailAddress, "@googlemail.com"))

	if isGmailAccount {
		// Gmail账户：使用统一的Gmail API同步方法
		s.logger.Debug("Detected Gmail account, using unified Gmail API sync")

		// 为Gmail账户移除日期过滤器，让Gmail History API自己处理增量同步
		gmailOptions := options
		gmailOptions.StartDate = nil

		// 直接调用Gmail API统一同步方法
		return s.fetchEmailsFromGmailAPI(account, gmailOptions)
	}

	// 非Gmail账户：使用传统的按文件夹分别同步方法
	s.logger.Info("Non-Gmail account, using traditional folder-by-folder sync")

	var emails []models.Email

	// Check if specific folders are provided in options
	if len(options.Folders) > 0 {
		// Fetch from user-selected folders
		s.logger.Debug("Fetching from user-selected folders: %v", options.Folders)
		for _, folder := range options.Folders {
			folderOptions := options
			folderOptions.Mailbox = folder

			s.logger.Debug("Fetching from folder: %s", folder)
			folderEmails, err := s.FetchEmailsWithOptions(account, folderOptions)
			if err != nil {
				s.logger.Error("Error fetching from folder %s: %v", folder, err)
				// Continue with other folders even if one fails
				continue
			}

			emails = append(emails, folderEmails...)
			s.logger.Info("Successfully fetched %d emails from %s", len(folderEmails), folder)
		}
	} else {
		// Fallback to fetching from the primary mailbox only
		s.logger.Debug("No specific folders provided, fetching from primary mailbox: %s", options.Mailbox)
		primaryEmails, err := s.FetchEmailsWithOptions(account, options)
		if err != nil {
			s.logger.Error("Error fetching from primary mailbox: %v", err)
			return nil, err
		}
		emails = primaryEmails
		s.logger.Info("Successfully fetched %d emails from primary mailbox", len(emails))
	}

	// Remove duplicates and apply date filter
	uniqueEmails := make(map[string]models.Email)
	filteredCount := 0

	for _, email := range emails {
		// Apply date filter if specified
		if options.StartDate != nil {
			// 确保两个时间都使用 UTC 进行比较
			emailDateUTC := email.Date.UTC()
			startDateUTC := options.StartDate.UTC()

			if emailDateUTC.Before(startDateUTC) {
				s.logger.Debug("Filtering out email - Subject: '%s', Date: %s (UTC) is before StartDate: %s (UTC)",
					email.Subject,
					emailDateUTC.Format(time.RFC3339),
					startDateUTC.Format(time.RFC3339))
				filteredCount++
				continue
			}

			// Log emails that pass the filter for debugging
			s.logger.Debug("Email passed date filter - Subject: '%s', Date: %s (UTC) >= StartDate: %s (UTC)",
				email.Subject,
				emailDateUTC.Format(time.RFC3339),
				startDateUTC.Format(time.RFC3339))
		}

		// Use MessageID as unique key, fallback to Subject+Date if MessageID is empty
		key := email.MessageID
		if key == "" {
			key = fmt.Sprintf("%s_%s", email.Subject, email.Date.Format(time.RFC3339))
		}
		uniqueEmails[key] = email
	}

	// Convert map back to slice
	result := make([]models.Email, 0, len(uniqueEmails))
	for _, email := range uniqueEmails {
		result = append(result, email)
	}

	// Sort by date (newest first)
	if len(result) > 1 {
		for i := 0; i < len(result)-1; i++ {
			for j := i + 1; j < len(result); j++ {
				if result[i].Date.Before(result[j].Date) {
					result[i], result[j] = result[j], result[i]
				}
			}
		}
	}

	s.logger.Info("Total emails fetched from all mailboxes: %d, filtered out: %d, unique emails: %d",
		len(emails), filteredCount, len(result))

	return result, nil
}

// fetchEmailsFromDatabase fetches emails from the local database
func (s *FetcherService) fetchEmailsFromDatabase(accountID uint, options FetchEmailsOptions) ([]models.Email, error) {
	s.logger.Debug("Fetching emails from database with options: %+v", options)

	// Convert sort option to SQL order clause
	sortClause := s.convertSortOption(options.SortBy)

	// Handle date range filtering
	if options.StartDate != nil && options.EndDate != nil {
		s.logger.Debug("Fetching by date range: %s to %s", options.StartDate.Format(time.RFC3339), options.EndDate.Format(time.RFC3339))
		return s.emailRepo.GetByDateRange(accountID, *options.StartDate, *options.EndDate)
	}

	// Handle search query
	if options.SearchQuery != "" {
		s.logger.Debug("Searching emails with query: %s", options.SearchQuery)
		return s.emailRepo.Search(accountID, options.SearchQuery)
	}

	// Handle mailbox filtering
	if options.Mailbox != "" && options.Mailbox != "INBOX" {
		s.logger.Debug("Fetching from specific mailbox: %s", options.Mailbox)
		return s.emailRepo.GetByAccountAndMailboxWithSort(accountID, options.Mailbox, options.Limit, options.Offset, sortClause)
	}

	// Default: get by account with pagination and sorting
	s.logger.Debug("Fetching with default pagination: limit=%d, offset=%d, sort=%s", options.Limit, options.Offset, sortClause)
	return s.emailRepo.GetByAccountWithSort(accountID, options.Limit, options.Offset, sortClause)
}

// convertSortOption converts API sort option to SQL order clause
func (s *FetcherService) convertSortOption(sortBy string) string {
	switch sortBy {
	case "date_asc":
		return "date ASC"
	case "subject_asc":
		return "subject ASC"
	case "subject_desc":
		return "subject DESC"
	case "date_desc":
		fallthrough
	default:
		return "date DESC"
	}
}

// fetchEmailsFromServer fetches emails from IMAP server with options
func (s *FetcherService) fetchEmailsFromServer(account models.EmailAccount, options FetchEmailsOptions) ([]models.Email, error) {
	// Check if should use Gmail API instead of IMAP
	if s.shouldUseGmailAPI(account) {
		s.logger.Debug("Using Gmail API for account %s", account.EmailAddress)
		return s.fetchEmailsFromGmailAPI(account, options)
	}

	var c *client.Client
	var err error

	// Check if MailProvider is nil
	if account.MailProvider == nil {
		return nil, fmt.Errorf("mail provider is not configured for account %s", account.EmailAddress)
	}

	serverAddr := fmt.Sprintf("%s:%d", account.MailProvider.IMAPServer, account.MailProvider.IMAPPort)
	s.logger.Info("Connecting to IMAP server %s for %s", serverAddr, account.EmailAddress)

	if account.Proxy != "" {
		proxyURL, err := url.Parse(account.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		dialer, err := s.createProxyDialer(proxyURL)
		if err != nil {
			s.logger.Error("Failed to create proxy dialer: %v", err)
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}

		s.logger.Debug("Connecting via %s proxy: %s", proxyURL.Scheme, account.Proxy)

		// For IMAP over proxy, we need to handle TLS after CONNECT
		if account.MailProvider.IMAPPort == 993 {
			// First establish the proxy tunnel
			proxyConn, err := dialer.Dial("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}

			// Then wrap with TLS
			s.logger.Debug("Establishing TLS connection through proxy tunnel")
			tlsConn := tls.Client(proxyConn, &tls.Config{
				ServerName: account.MailProvider.IMAPServer,
			})

			// Perform TLS handshake
			if err := tlsConn.Handshake(); err != nil {
				proxyConn.Close()
				s.logger.Error("TLS handshake failed: %v", err)
				return nil, fmt.Errorf("TLS handshake failed: %w", err)
			}

			// Create IMAP client with the TLS connection
			c, err = client.New(tlsConn)
			if err != nil {
				tlsConn.Close()
				s.logger.Error("Failed to create IMAP client: %v", err)
				return nil, fmt.Errorf("failed to create IMAP client: %w", err)
			}
		} else {
			// For non-TLS IMAP, use the proxy connection directly
			c, err = client.DialWithDialer(dialer, serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}
		}
	} else {
		// Use TLS connection for secure IMAP (port 993)
		if account.MailProvider.IMAPPort == 993 {
			s.logger.Debug("Using TLS connection for port 993")
			c, err = client.DialTLS(serverAddr, &tls.Config{ServerName: account.MailProvider.IMAPServer})
			if err != nil {
				s.logger.Error("Failed to dial with TLS: %v", err)
				return nil, fmt.Errorf("failed to dial with TLS: %w", err)
			}
		} else {
			// Use plain connection for non-secure IMAP (port 143)
			s.logger.Debug("Using plain connection for port %d", account.MailProvider.IMAPPort)
			c, err = client.Dial(serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial: %v", err)
				return nil, fmt.Errorf("failed to dial: %w", err)
			}
		}
	}
	defer c.Logout()

	// Login based on auth type
	s.logger.Debug("Authenticating with auth type: %s", account.AuthType)
	switch account.AuthType {
	case models.AuthTypePassword:
		// Standard password authentication
		if err := c.Login(account.EmailAddress, account.Password); err != nil {
			s.logger.Error("Password authentication failed for %s: %v", account.EmailAddress, err)
			return nil, fmt.Errorf("login failed: %w", err)
		}
	case models.AuthTypeOAuth2:
		// OAuth2 authentication
		s.logger.Debug("Using OAuth2 authentication")
		// Get client_id from CustomSettings, with fallback to global config
		clientID, ok := account.CustomSettings["client_id"]
		if !ok {
			s.logger.Warn("client_id not found in custom settings, trying to get from global config")

			// Try to get client_id from global OAuth2 config
			oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())
			var config *models.OAuth2GlobalConfig
			var err error

			// First try by OAuth2ProviderID if available
			if account.OAuth2ProviderID != nil {
				config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
				if err != nil {
					s.logger.Warn("Failed to get config by OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
				}
			}

			// Fallback to provider type
			if config == nil {
				config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
				if err != nil {
					s.logger.Error("Failed to get global config for provider %s: %v", account.MailProvider.Type, err)
					return nil, fmt.Errorf("client_id not found in custom settings and failed to get from global config: %w", err)
				}
			}

			if config == nil {
				return nil, fmt.Errorf("client_id not found in custom settings and no global config available for provider %s", account.MailProvider.Type)
			}

			clientID = config.ClientID
			s.logger.Info("Using client_id from global config (ID: %d, Name: %s) for fetchEmailsFromServer", config.ID, config.Name)
		}

		refreshToken, ok := account.CustomSettings["refresh_token"]
		if !ok {
			s.logger.Error("refresh_token not found in custom settings")
			return nil, fmt.Errorf("refresh_token not found in custom settings")
		}

		// Get client_secret from global OAuth2 config (secure approach)
		clientSecret := ""
		oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

		var config *models.OAuth2GlobalConfig
		var err error

		// Priority 1: Use OAuth2ProviderID if available (new multi-config support)
		if account.OAuth2ProviderID != nil && *account.OAuth2ProviderID > 0 {
			s.logger.Debug("Using OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get config from OAuth2ProviderID %d for account %s: %v", *account.OAuth2ProviderID, account.EmailAddress, err)
			}
		}

		// Priority 2: Fallback to provider type lookup (backward compatibility)
		if config == nil {
			s.logger.Debug("Falling back to provider type lookup for %s", account.MailProvider.Type)
			config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from provider type %s for account %s", account.MailProvider.Type, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get client_secret from provider type %s for account %s: %v", account.MailProvider.Type, account.EmailAddress, err)
			}
		}

		// Refresh access token - use cached method with retry protection and proxy support
		s.logger.Debug("Refreshing OAuth2 access token with cache for provider: %s", account.MailProvider.Type)
		accessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
			string(account.MailProvider.Type),
			clientID,
			clientSecret,
			refreshToken,
			account.ID,
			account.Proxy, // Pass proxy settings if available
		)
		if err != nil {
			s.logger.Error("Failed to refresh access token: %v", err)
			return nil, fmt.Errorf("failed to refresh access token: %w", err)
		}

		// Update access token in account - 创建新的副本以避免并发写入问题
		newCustomSettings := make(models.JSONMap)
		if account.CustomSettings != nil {
			// 复制现有设置
			for k, v := range account.CustomSettings {
				newCustomSettings[k] = v
			}
		}
		newCustomSettings["access_token"] = accessToken
		account.CustomSettings = newCustomSettings

		// Update the account with new access token
		updatedAccount := account
		if err := s.accountRepo.Update(&updatedAccount); err != nil {
			s.logger.Warn("Failed to update access token in database: %v", err)
		}

		// Authenticate with OAuth2
		saslClient := NewOAuth2SASLClient(account.EmailAddress, accessToken)
		if err := c.Authenticate(saslClient); err != nil {
			s.logger.Error("OAuth2 authentication failed: %v", err)
			return nil, fmt.Errorf("OAuth2 authentication failed: %w", err)
		}
	default:
		s.logger.Error("Unsupported auth type: %s", account.AuthType)
		return nil, fmt.Errorf("unsupported auth type: %s", account.AuthType)
	}

	s.logger.Info("Successfully connected and logged in for %s using %s auth", account.EmailAddress, account.AuthType)

	// Check connection state after authentication
	if c.State() != imap.AuthenticatedState && c.State() != imap.SelectedState {
		s.logger.Error("IMAP connection is in unexpected state after authentication: %s", c.State())
		return nil, fmt.Errorf("IMAP connection is in unexpected state after authentication: %s", c.State())
	}

	s.logger.Debug("IMAP connection state after authentication: %s", c.State())

	// Select mailbox (default to INBOX if not specified)
	mailboxName := options.Mailbox
	if mailboxName == "" {
		mailboxName = "INBOX"
	}

	s.logger.Debug("Attempting to select mailbox: %s", mailboxName)
	mbox, err := c.Select(mailboxName, false)
	if err != nil {
		s.logger.Error("Failed to select mailbox %s: %v (connection state: %s)", mailboxName, err, c.State())

		// Try to check if connection is still alive
		if c.State() == imap.LogoutState || c.State() == imap.NotAuthenticatedState {
			s.logger.Error("Connection appears to be disconnected, state: %s", c.State())
			return nil, fmt.Errorf("connection lost after authentication, failed to select mailbox %s: %w", mailboxName, err)
		}

		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailboxName, err)
	}

	s.logger.Info("Selected mailbox %s: %d total messages", mailboxName, mbox.Messages)

	// Calculate message range based on limit and offset
	limit := options.Limit
	if limit <= 0 || limit > 100 {
		limit = 10 // Default limit
	}

	offset := options.Offset
	if offset < 0 {
		offset = 0
	}

	// Calculate from and to based on total messages, limit, and offset
	from := uint32(1)
	to := mbox.Messages

	if mbox.Messages > 0 {
		// For IMAP, we need to calculate from the end since we want recent emails
		if int(mbox.Messages) > offset+limit {
			from = mbox.Messages - uint32(offset+limit-1)
			to = mbox.Messages - uint32(offset)
		} else if int(mbox.Messages) > offset {
			from = uint32(1)
			to = mbox.Messages - uint32(offset)
		} else {
			// No messages in the requested range
			return []models.Email{}, nil
		}
	}

	// Apply date filter using IMAP search if start date is specified
	var seqNums []uint32
	if options.StartDate != nil {
		s.logger.Debug("Searching for emails since %s in mailbox %s", options.StartDate.Format("02-Jan-2006"), mailboxName)

		criteria := imap.NewSearchCriteria()
		criteria.Since = *options.StartDate

		seqNums, err = c.Search(criteria)
		if err != nil {
			s.logger.Error("Failed to search messages: %v", err)
			return nil, fmt.Errorf("failed to search messages: %w", err)
		}

		s.logger.Info("Found %d messages matching search criteria in %s", len(seqNums), mailboxName)

		if len(seqNums) == 0 {
			return []models.Email{}, nil
		}

		// Limit the results if needed
		if len(seqNums) > limit {
			// Get the most recent messages
			seqNums = seqNums[len(seqNums)-limit:]
		}
	} else {
		// No date filter, use the original range
		for i := from; i <= to; i++ {
			seqNums = append(seqNums, i)
		}
	}

	seqset := new(imap.SeqSet)
	for _, num := range seqNums {
		seqset.AddNum(num)
	}

	// Prepare fetch items based on options
	fetchItems := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size}
	if options.IncludeBody {
		fetchItems = append(fetchItems, imap.FetchRFC822)
	}

	// Fetch messages
	messages := make(chan *imap.Message, limit)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, fetchItems, messages)
	}()

	var emails []models.Email
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}

		email := models.Email{
			MessageID:   msg.Envelope.MessageId,
			AccountID:   account.ID,
			Subject:     msg.Envelope.Subject,
			Date:        msg.Envelope.Date,
			MailboxName: mailboxName,
			Size:        int64(msg.Size),
		}

		// Convert addresses
		email.From = convertAddresses(msg.Envelope.From)
		email.To = convertAddresses(msg.Envelope.To)
		email.Cc = convertAddresses(msg.Envelope.Cc)
		email.Bcc = convertAddresses(msg.Envelope.Bcc)

		// Convert flags
		for _, flag := range msg.Flags {
			email.Flags = append(email.Flags, string(flag))
		}

		// Parse email body content if available and requested
		if options.IncludeBody && msg.Body != nil && len(msg.Body) > 0 {
			for _, body := range msg.Body {
				if body != nil {
					// Read the raw email content
					rawEmail, err := ioutil.ReadAll(body)
					if err != nil {
						s.logger.Warn("Failed to read email body for message %s: %v", email.MessageID, err)
						continue
					}

					// Parse the email content using the parser service
					parsedEmail, err := s.parserService.ParseEmail(rawEmail)
					if err != nil {
						s.logger.Warn("Failed to parse email content for message %s: %v", email.MessageID, err)
						continue
					}

					// Update email with parsed content
					if parsedEmail.Body != "" {
						email.Body = parsedEmail.Body
					}
					if parsedEmail.HTMLBody != "" {
						email.HTMLBody = parsedEmail.HTMLBody
					}
					if len(parsedEmail.Attachments) > 0 {
						email.Attachments = parsedEmail.Attachments
					}
					break // Only process the first body part
				}
			}
		}

		emails = append(emails, email)

		s.logger.Debug("Fetched email %d - Subject: '%s', From: %s, Date: %s",
			len(emails), email.Subject, email.From, email.Date.Format(time.RFC3339))
	}

	if err := <-done; err != nil {
		s.logger.Error("Failed to fetch messages: %v", err)
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Update last sync time
	if err := s.accountRepo.UpdateLastSync(account.ID); err != nil {
		s.logger.Warn("Failed to update last sync time: %v", err)
	}

	s.logger.Info("Successfully fetched %d emails from %s", len(emails), mailboxName)
	return emails, nil
}

// FetchEmailsByAccountID fetches emails for a given account ID
func (s *FetcherService) FetchEmailsByAccountID(accountID uint) ([]models.Email, error) {
	s.logger.Debug("FetchEmailsByAccountID called for account ID: %d", accountID)

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		s.logger.Error("Failed to get account %d: %v", accountID, err)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return s.FetchEmails(*account)
}

// FetchAndStoreEmails fetches emails and stores them in the database
func (s *FetcherService) FetchAndStoreEmails(accountID uint) error {
	s.logger.Info("FetchAndStoreEmails called for account ID: %d", accountID)

	emails, err := s.FetchEmailsByAccountID(accountID)
	if err != nil {
		s.logger.Error("Failed to fetch emails for account %d: %v", accountID, err)
		return err
	}

	s.logger.Debug("Fetched %d emails, checking for duplicates", len(emails))

	// Check for duplicates and store new emails
	var newEmails []models.Email
	duplicateCount := 0
	for _, email := range emails {
		if email.MessageID != "" {
			exists, err := s.emailRepo.CheckDuplicate(email.MessageID, accountID)
			if err != nil {
				s.logger.Warn("Error checking duplicate for message %s: %v", email.MessageID, err)
				continue
			}
			if exists {
				duplicateCount++
				continue
			}
		}
		newEmails = append(newEmails, email)
	}

	s.logger.Debug("Found %d duplicates out of %d emails", duplicateCount, len(emails))

	if len(newEmails) > 0 {
		if err := s.emailRepo.CreateBatch(newEmails); err != nil {
			s.logger.Error("Failed to store emails: %v", err)
			return fmt.Errorf("failed to store emails: %w", err)
		}
		s.logger.Info("Stored %d new emails for account %d", len(newEmails), accountID)
	} else {
		s.logger.Info("No new emails to store for account %d", accountID)
	}

	return nil
}

// GetMailboxes retrieves all mailboxes for an account
func (s *FetcherService) GetMailboxes(account models.EmailAccount) ([]models.Mailbox, error) {
	s.logger.Info("GetMailboxes called for account %s", account.EmailAddress)

	// Check if MailProvider is nil
	if account.MailProvider == nil {
		return nil, fmt.Errorf("mail provider is not configured for account %s", account.EmailAddress)
	}

	// For Gmail OAuth2 accounts, use Gmail API instead of IMAP
	if s.shouldUseGmailAPI(account) {
		s.logger.Debug("Using Gmail API to get mailboxes for OAuth2 account")
		return s.getGmailMailboxes(account)
	}

	// For other accounts, use IMAP
	var c *client.Client
	var err error

	serverAddr := fmt.Sprintf("%s:%d", account.MailProvider.IMAPServer, account.MailProvider.IMAPPort)

	if account.Proxy != "" {
		proxyURL, err := url.Parse(account.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		dialer, err := s.createProxyDialer(proxyURL)
		if err != nil {
			s.logger.Error("Failed to create proxy dialer: %v", err)
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}

		s.logger.Debug("Connecting via %s proxy: %s", proxyURL.Scheme, account.Proxy)

		// For IMAP over proxy, we need to handle TLS after CONNECT
		if account.MailProvider.IMAPPort == 993 {
			// First establish the proxy tunnel
			proxyConn, err := dialer.Dial("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}

			// Then wrap with TLS
			s.logger.Debug("Establishing TLS connection through proxy tunnel")
			tlsConn := tls.Client(proxyConn, &tls.Config{
				ServerName: account.MailProvider.IMAPServer,
			})

			// Perform TLS handshake
			if err := tlsConn.Handshake(); err != nil {
				proxyConn.Close()
				s.logger.Error("TLS handshake failed: %v", err)
				return nil, fmt.Errorf("TLS handshake failed: %w", err)
			}

			// Create IMAP client with the TLS connection
			c, err = client.New(tlsConn)
			if err != nil {
				tlsConn.Close()
				s.logger.Error("Failed to create IMAP client: %v", err)
				return nil, fmt.Errorf("failed to create IMAP client: %w", err)
			}
		} else {
			// For non-TLS IMAP, use the proxy connection directly
			c, err = client.DialWithDialer(dialer, serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}
		}
	} else {
		// Use TLS connection for secure IMAP (port 993)
		if account.MailProvider.IMAPPort == 993 {
			s.logger.Debug("Using TLS connection for port 993")
			c, err = client.DialTLS(serverAddr, &tls.Config{ServerName: account.MailProvider.IMAPServer})
			if err != nil {
				s.logger.Error("Failed to dial with TLS: %v", err)
				return nil, fmt.Errorf("failed to dial with TLS: %w", err)
			}
		} else {
			// Use plain connection for non-secure IMAP (port 143)
			s.logger.Debug("Using plain connection for port %d", account.MailProvider.IMAPPort)
			c, err = client.Dial(serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial: %v", err)
				return nil, fmt.Errorf("failed to dial: %w", err)
			}
		}
	}
	defer c.Logout()

	// Login based on auth type
	s.logger.Debug("Authenticating with auth type: %s", account.AuthType)
	switch account.AuthType {
	case models.AuthTypePassword:
		// Standard password authentication
		if err := c.Login(account.EmailAddress, account.Password); err != nil {
			s.logger.Error("Password authentication failed: %v", err)
			return nil, fmt.Errorf("login failed: %w", err)
		}
	case models.AuthTypeOAuth2:
		// OAuth2 authentication
		s.logger.Debug("Using OAuth2 authentication")
		// Get client_id and refresh_token from CustomSettings
		clientID, ok := account.CustomSettings["client_id"]
		if !ok {
			s.logger.Error("client_id not found in custom settings")
			return nil, fmt.Errorf("client_id not found in custom settings")
		}

		refreshToken, ok := account.CustomSettings["refresh_token"]
		if !ok {
			s.logger.Error("refresh_token not found in custom settings")
			return nil, fmt.Errorf("refresh_token not found in custom settings")
		}

		// Get client_secret from global OAuth2 config (secure approach)
		clientSecret := ""
		oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

		var config *models.OAuth2GlobalConfig
		var err error

		// Priority 1: Use OAuth2ProviderID if available (new multi-config support)
		if account.OAuth2ProviderID != nil && *account.OAuth2ProviderID > 0 {
			s.logger.Debug("Using OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get config from OAuth2ProviderID %d for account %s: %v", *account.OAuth2ProviderID, account.EmailAddress, err)
			}
		}

		// Priority 2: Fallback to provider type lookup (backward compatibility)
		if config == nil {
			s.logger.Debug("Falling back to provider type lookup for %s", account.MailProvider.Type)
			config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from provider type %s for account %s", account.MailProvider.Type, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get client_secret from provider type %s for account %s: %v", account.MailProvider.Type, account.EmailAddress, err)
			}
		}

		// Refresh access token - use cached method with concurrency protection for better reliability
		s.logger.Debug("Refreshing OAuth2 access token for IMAP connection with cache")
		accessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
			string(account.MailProvider.Type),
			clientID,
			clientSecret,
			refreshToken,
			account.ID,
			account.Proxy, // Pass proxy settings if available
		)
		if err != nil {
			s.logger.Error("Failed to refresh access token: %v", err)
			return nil, fmt.Errorf("failed to refresh access token: %w", err)
		}

		// Update access token in account
		if account.CustomSettings == nil {
			account.CustomSettings = make(models.JSONMap)
		}
		// 并发安全的CustomSettings更新
		newCustomSettings := make(models.JSONMap)
		if account.CustomSettings != nil {
			for k, v := range account.CustomSettings {
				newCustomSettings[k] = v
			}
		}
		newCustomSettings["access_token"] = accessToken
		account.CustomSettings = newCustomSettings

		// Update the account with new access token
		updatedAccount := account
		if err := s.accountRepo.Update(&updatedAccount); err != nil {
			s.logger.Warn("Failed to update access token in database: %v", err)
		}

		// Authenticate with OAuth2
		saslClient := NewOAuth2SASLClient(account.EmailAddress, accessToken)
		if err := c.Authenticate(saslClient); err != nil {
			s.logger.Error("OAuth2 authentication failed: %v", err)
			return nil, fmt.Errorf("OAuth2 authentication failed: %w", err)
		}
	default:
		s.logger.Error("Unsupported auth type: %s", account.AuthType)
		return nil, fmt.Errorf("unsupported auth type: %s", account.AuthType)
	}

	s.logger.Debug("Successfully authenticated, listing mailboxes")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var mboxes []models.Mailbox
	for mbox := range mailboxes {
		mailbox := models.Mailbox{
			Name:      mbox.Name,
			AccountID: account.ID,
			Delimiter: mbox.Delimiter,
		}

		// Convert attributes to flags
		for _, attr := range mbox.Attributes {
			mailbox.Flags = append(mailbox.Flags, string(attr))
		}

		mboxes = append(mboxes, mailbox)
	}

	if err := <-done; err != nil {
		s.logger.Error("Failed to list mailboxes: %v", err)
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	s.logger.Info("Successfully retrieved %d mailboxes for account %s", len(mboxes), account.EmailAddress)
	return mboxes, nil
}

// createProxyDialer creates a dialer based on the proxy URL scheme
func (s *FetcherService) createProxyDialer(proxyURL *url.URL) (proxy.Dialer, error) {
	switch proxyURL.Scheme {
	case "socks5", "socks5h":
		// Use the existing SOCKS5 support
		return proxy.FromURL(proxyURL, proxy.Direct)
	case "http", "https":
		// Create HTTP proxy dialer
		return s.createHTTPProxyDialer(proxyURL), nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}
}

// createHTTPProxyDialer creates a dialer for HTTP/HTTPS proxy
func (s *FetcherService) createHTTPProxyDialer(proxyURL *url.URL) proxy.Dialer {
	return &httpProxyDialer{
		proxyURL: proxyURL,
		logger:   s.logger,
	}
}

// httpProxyDialer implements proxy.Dialer for HTTP/HTTPS proxies
type httpProxyDialer struct {
	proxyURL *url.URL
	logger   *utils.Logger
}

// createHTTPClientWithProxy creates an HTTP client with proxy support
func (s *FetcherService) createHTTPClientWithProxy(proxyStr string) (*http.Client, error) {
	if proxyStr == "" {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	s.logger.Debug("Creating HTTP client with proxy: %s", proxyStr)

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

// Dial implements the proxy.Dialer interface
func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	d.logger.Debug("HTTP proxy dialer: attempting to connect to %s via proxy %s", addr, d.proxyURL.Host)

	// Connect to the proxy server
	proxyHost := d.proxyURL.Host
	if proxyHost == "" {
		return nil, fmt.Errorf("proxy URL missing host")
	}

	// Add default port if not specified
	if !strings.Contains(proxyHost, ":") {
		if d.proxyURL.Scheme == "https" {
			proxyHost += ":443"
		} else {
			proxyHost += ":80"
		}
	}

	d.logger.Debug("Connecting to proxy server at %s", proxyHost)

	// Connect to proxy with timeout
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	var proxyConn net.Conn
	var err error

	// For HTTPS proxy, establish TLS connection
	if d.proxyURL.Scheme == "https" {
		d.logger.Debug("Using TLS connection for HTTPS proxy")
		proxyConn, err = tls.DialWithDialer(dialer, "tcp", proxyHost, &tls.Config{
			ServerName: strings.Split(proxyHost, ":")[0],
		})
	} else {
		proxyConn, err = dialer.Dial("tcp", proxyHost)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy at %s: %w", proxyHost, err)
	}

	// Ensure we have the target host and port
	targetHost, targetPort, err := net.SplitHostPort(addr)
	if err != nil {
		// If no port specified, assume IMAP default ports
		targetHost = addr
		targetPort = "993" // Default IMAP SSL port
		addr = net.JoinHostPort(targetHost, targetPort)
	}

	// Create CONNECT request
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n", addr)
	connectReq += fmt.Sprintf("Host: %s\r\n", addr)
	connectReq += "User-Agent: mailman/1.0\r\n"
	connectReq += "Proxy-Connection: Keep-Alive\r\n"

	// Add proxy authentication if provided
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth := username + ":" + password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", encodedAuth)
		d.logger.Debug("Adding proxy authentication for user: %s", username)
	}

	connectReq += "\r\n"

	d.logger.Debug("Sending CONNECT request to proxy:\n%s", strings.ReplaceAll(connectReq, "\r\n", "\\r\\n"))

	// Send CONNECT request
	_, err = proxyConn.Write([]byte(connectReq))
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	// Read response with timeout
	proxyConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(proxyConn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		proxyConn.Close()
		if err == io.EOF {
			return nil, fmt.Errorf("proxy closed connection unexpectedly (EOF) - proxy may not support CONNECT method or requires authentication")
		}
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}
	proxyConn.SetReadDeadline(time.Time{}) // Clear deadline

	d.logger.Debug("Proxy response: %s", strings.TrimSpace(statusLine))

	// Parse status code
	parts := strings.Fields(statusLine)
	if len(parts) < 2 {
		proxyConn.Close()
		return nil, fmt.Errorf("invalid proxy response: %s", statusLine)
	}

	statusCode := parts[1]
	if statusCode != "200" {
		// Read the rest of the response for error details
		var responseBody strings.Builder
		responseBody.WriteString(statusLine)

		// Read headers
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			responseBody.WriteString(line)
			if line == "\r\n" || line == "\n" {
				break
			}
		}

		proxyConn.Close()

		// Provide specific error messages for common status codes
		switch statusCode {
		case "407":
			return nil, fmt.Errorf("proxy authentication required (407) - please provide valid proxy credentials")
		case "403":
			return nil, fmt.Errorf("proxy access forbidden (403) - the proxy server rejected the connection")
		case "502":
			return nil, fmt.Errorf("bad gateway (502) - the proxy server received an invalid response from the upstream server")
		case "503":
			return nil, fmt.Errorf("service unavailable (503) - the proxy server is temporarily unable to handle the request")
		default:
			return nil, fmt.Errorf("proxy connection failed with status %s: %s", statusCode, strings.TrimSpace(responseBody.String()))
		}
	}

	// Read and discard headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			proxyConn.Close()
			return nil, fmt.Errorf("failed to read proxy headers: %w", err)
		}
		d.logger.Debug("Proxy header: %s", strings.TrimSpace(line))
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	d.logger.Info("Successfully established connection through HTTP proxy to %s", addr)
	return proxyConn, nil
}

// convertAddresses converts IMAP addresses to string slice
func convertAddresses(addresses []*imap.Address) models.StringSlice {
	var result models.StringSlice
	for _, addr := range addresses {
		if addr != nil {
			// Format email address properly
			emailAddr := fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)

			// If there's a personal name, format as "Name <email>"
			if addr.PersonalName != "" {
				result = append(result, fmt.Sprintf("%s <%s>", addr.PersonalName, emailAddr))
			} else {
				// Otherwise, just the email address
				result = append(result, emailAddr)
			}
		}
	}
	return result
}

// VerifyConnection verifies if an email account can connect successfully
func (s *FetcherService) VerifyConnection(account models.EmailAccount) error {
	s.logger.Info("VerifyConnection called for account %s", account.EmailAddress)

	// Check if MailProvider is nil
	if account.MailProvider == nil {
		return fmt.Errorf("mail provider is not configured for account %s", account.EmailAddress)
	}

	// For Gmail OAuth2 accounts, use Gmail API instead of IMAP
	if account.AuthType == models.AuthTypeOAuth2 && account.MailProvider.Type == models.ProviderTypeGmail {
		s.logger.Debug("Using Gmail API verification for OAuth2 account")
		return s.verifyGmailOAuth2Connection(account)
	}

	// For other accounts, use IMAP verification
	var c *client.Client
	var err error

	serverAddr := fmt.Sprintf("%s:%d", account.MailProvider.IMAPServer, account.MailProvider.IMAPPort)
	s.logger.Debug("Connecting to IMAP server %s", serverAddr)

	// Establish connection
	if account.Proxy != "" {
		proxyURL, err := url.Parse(account.Proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}

		dialer, err := s.createProxyDialer(proxyURL)
		if err != nil {
			s.logger.Error("Failed to create proxy dialer: %v", err)
			return fmt.Errorf("failed to create proxy dialer: %w", err)
		}

		s.logger.Debug("Connecting via %s proxy: %s", proxyURL.Scheme, account.Proxy)

		// For IMAP over proxy, we need to handle TLS after CONNECT
		if account.MailProvider.IMAPPort == 993 {
			// First establish the proxy tunnel
			proxyConn, err := dialer.Dial("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return fmt.Errorf("failed to dial via proxy: %w", err)
			}

			// Then wrap with TLS
			s.logger.Debug("Establishing TLS connection through proxy tunnel")
			tlsConn := tls.Client(proxyConn, &tls.Config{
				ServerName: account.MailProvider.IMAPServer,
			})

			// Perform TLS handshake
			if err := tlsConn.Handshake(); err != nil {
				proxyConn.Close()
				s.logger.Error("TLS handshake failed: %v", err)
				return fmt.Errorf("TLS handshake failed: %w", err)
			}

			// Create IMAP client with the TLS connection
			c, err = client.New(tlsConn)
			if err != nil {
				tlsConn.Close()
				s.logger.Error("Failed to create IMAP client: %v", err)
				return fmt.Errorf("failed to create IMAP client: %w", err)
			}
		} else {
			// For non-TLS IMAP, use the proxy connection directly
			c, err = client.DialWithDialer(dialer, serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return fmt.Errorf("failed to dial via proxy: %w", err)
			}
		}
	} else {
		// Use TLS connection for secure IMAP (port 993)
		if account.MailProvider.IMAPPort == 993 {
			s.logger.Debug("Using TLS connection for port 993")
			c, err = client.DialTLS(serverAddr, &tls.Config{ServerName: account.MailProvider.IMAPServer})
			if err != nil {
				s.logger.Error("Failed to dial with TLS: %v", err)
				return fmt.Errorf("failed to dial with TLS: %w", err)
			}
		} else {
			// Use plain connection for non-secure IMAP (port 143)
			s.logger.Debug("Using plain connection for port %d", account.MailProvider.IMAPPort)
			c, err = client.Dial(serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial: %v", err)
				return fmt.Errorf("failed to dial: %w", err)
			}
		}
	}
	defer c.Logout()

	// Login based on auth type
	s.logger.Debug("Authenticating with auth type: %s", account.AuthType)
	switch account.AuthType {
	case models.AuthTypePassword:
		// Standard password authentication
		if err := c.Login(account.EmailAddress, account.Password); err != nil {
			s.logger.Error("Password authentication failed: %v", err)
			return fmt.Errorf("login failed: %w", err)
		}
	case models.AuthTypeOAuth2:
		// OAuth2 authentication
		s.logger.Debug("Using OAuth2 authentication")
		s.logger.Debug("CustomSettings content: %+v", account.CustomSettings)
		// Get client_id from CustomSettings, with fallback to global config
		clientID, ok := account.CustomSettings["client_id"]
		if !ok {
			s.logger.Warn("client_id not found in custom settings, trying to get from global config")
			s.logger.Debug("Available keys in CustomSettings: %v", func() []string {
				keys := make([]string, 0, len(account.CustomSettings))
				for k := range account.CustomSettings {
					keys = append(keys, k)
				}
				return keys
			}())

			// Try to get client_id from global OAuth2 config
			oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())
			var config *models.OAuth2GlobalConfig
			var err error

			// First try by OAuth2ProviderID if available
			if account.OAuth2ProviderID != nil {
				config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
				if err != nil {
					s.logger.Warn("Failed to get config by OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
				}
			}

			// Fallback to provider type
			if config == nil {
				config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
				if err != nil {
					s.logger.Error("Failed to get global config for provider %s: %v", account.MailProvider.Type, err)
					return fmt.Errorf("client_id not found in custom settings and failed to get from global config: %w", err)
				}
			}

			if config == nil {
				return fmt.Errorf("client_id not found in custom settings and no global config available for provider %s", account.MailProvider.Type)
			}

			clientID = config.ClientID
			s.logger.Info("Using client_id from global config (ID: %d, Name: %s) for account %s", config.ID, config.Name, account.EmailAddress)
		}

		refreshToken, ok := account.CustomSettings["refresh_token"]
		if !ok {
			s.logger.Error("refresh_token not found in custom settings")
			return fmt.Errorf("refresh_token not found in custom settings")
		}

		// Get client_secret from global OAuth2 config (secure approach)
		clientSecret := ""
		oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

		var config *models.OAuth2GlobalConfig
		var err error

		// First try to get config by OAuth2ProviderID if it exists
		if account.OAuth2ProviderID != nil {
			s.logger.Debug("Using OAuth2ProviderID %d to get config", *account.OAuth2ProviderID)
			config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
			if err != nil {
				s.logger.Warn("Failed to get config by OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
			}
		}

		// Fallback to provider type based lookup for backward compatibility
		if config == nil {
			s.logger.Debug("Falling back to provider type based lookup for: %s", account.MailProvider.Type)
			config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
			if err != nil {
				s.logger.Warn("Failed to get client_secret from global config for provider %s: %v", account.MailProvider.Type, err)
			}
		}

		if config != nil {
			clientSecret = config.ClientSecret
			s.logger.Debug("Retrieved client_secret from global config (ID: %d, Name: %s)", config.ID, config.Name)
		}

		// Refresh access token - use cached method with concurrency protection for better reliability
		s.logger.Debug("Refreshing OAuth2 access token for IMAP connection with cache")
		accessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
			string(account.MailProvider.Type),
			clientID,
			clientSecret,
			refreshToken,
			account.ID,
			account.Proxy, // Pass proxy settings if available
		)
		if err != nil {
			s.logger.Error("Failed to refresh access token: %v", err)
			return fmt.Errorf("failed to refresh access token: %w", err)
		}

		// Authenticate with OAuth2
		saslClient := NewOAuth2SASLClient(account.EmailAddress, accessToken)
		if err := c.Authenticate(saslClient); err != nil {
			s.logger.Error("OAuth2 authentication failed: %v", err)
			return fmt.Errorf("OAuth2 authentication failed: %w", err)
		}
	default:
		s.logger.Error("Unsupported auth type: %s", account.AuthType)
		return fmt.Errorf("unsupported auth type: %s", account.AuthType)
	}

	// If we reach here, connection and authentication were successful
	s.logger.Info("Successfully verified connection for %s using %s auth", account.EmailAddress, account.AuthType)

	// Try to select INBOX to ensure full connectivity
	_, err = c.Select("INBOX", false)
	if err != nil {
		s.logger.Error("Failed to select INBOX: %v", err)
		return fmt.Errorf("failed to select INBOX: %w", err)
	}

	s.logger.Info("Connection verification successful for %s", account.EmailAddress)
	return nil
}

// verifyGmailOAuth2Connection verifies Gmail OAuth2 connection using Gmail API
func (s *FetcherService) verifyGmailOAuth2Connection(account models.EmailAccount) error {
	s.logger.Info("Verifying Gmail OAuth2 connection for %s", account.EmailAddress)

	// Get OAuth2 configuration
	oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

	var oauth2Config *models.OAuth2GlobalConfig
	var err error

	// First try to get config by OAuth2ProviderID if it exists
	if account.OAuth2ProviderID != nil {
		s.logger.Debug("Using OAuth2ProviderID %d to get config for Gmail verification", *account.OAuth2ProviderID)
		oauth2Config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
		if err != nil {
			s.logger.Warn("Failed to get config by OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
		}
	}

	// Fallback to provider type based lookup for backward compatibility
	if oauth2Config == nil {
		s.logger.Debug("Falling back to provider type based lookup for Gmail")
		oauth2Config, err = oauth2GlobalConfigRepo.GetByProviderType(models.ProviderTypeGmail)
		if err != nil {
			s.logger.Error("Failed to get OAuth2 config: %v", err)
			return fmt.Errorf("failed to get OAuth2 config: %w", err)
		}
	}

	if oauth2Config == nil {
		s.logger.Error("No OAuth2 config found")
		return fmt.Errorf("no OAuth2 config found")
	}

	s.logger.Debug("Using OAuth2 config: ID=%d, Name=%s", oauth2Config.ID, oauth2Config.Name)

	// Get tokens from CustomSettings
	if account.CustomSettings == nil {
		s.logger.Error("CustomSettings is nil for account %s", account.EmailAddress)
		return fmt.Errorf("OAuth2 tokens not found")
	}

	accessToken, ok := account.CustomSettings["access_token"]
	if !ok || accessToken == "" {
		s.logger.Error("access_token not found in CustomSettings")
		return fmt.Errorf("access_token not found")
	}

	refreshToken, ok := account.CustomSettings["refresh_token"]
	if !ok || refreshToken == "" {
		s.logger.Error("refresh_token not found in CustomSettings")
		return fmt.Errorf("refresh_token not found")
	}

	// Try to refresh token first to ensure it's valid - use cached method for better reliability
	s.logger.Debug("Refreshing OAuth2 access token for Gmail verification")
	newAccessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
		"gmail",
		oauth2Config.ClientID,
		oauth2Config.ClientSecret,
		refreshToken,
		account.ID,
		account.Proxy, // Pass proxy settings if available
	)
	if err != nil {
		s.logger.Error("Failed to refresh token: %v", err)
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Use the refreshed token
	accessToken = newAccessToken
	s.logger.Debug("Token refreshed successfully")

	// Test connection by making a direct HTTP request to Gmail API
	s.logger.Debug("Testing Gmail API connection by getting labels list")
	req, err := http.NewRequest("GET", "https://gmail.googleapis.com/gmail/v1/users/me/labels", nil)
	if err != nil {
		s.logger.Error("Failed to create HTTP request: %v", err)
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to make HTTP request: %v", err)
		return fmt.Errorf("Gmail API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		s.logger.Error("Gmail API returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("Gmail API verification failed with status %d", resp.StatusCode)
	}

	// Parse the response to get labels count
	var labelsResponse struct {
		Labels []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, &labelsResponse)
	if err != nil {
		s.logger.Error("Failed to parse JSON response: %v", err)
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	s.logger.Info("Gmail OAuth2 connection verified successfully for %s, found %d labels", account.EmailAddress, len(labelsResponse.Labels))
	return nil
}

// GetAllFolders retrieves all available folders/labels for an email account
func (s *FetcherService) GetAllFolders(account models.EmailAccount) ([]string, error) {
	s.logger.Debug("Getting all folders for account %s", account.EmailAddress)

	// For Gmail OAuth2 accounts, use Gmail API to get labels
	if account.AuthType == models.AuthTypeOAuth2 && account.MailProvider.Type == models.ProviderTypeGmail {
		return s.getGmailFolders(account)
	}

	// For IMAP accounts, use IMAP LIST command
	return s.getImapFolders(account)
}

// getGmailFolders retrieves Gmail folders (labels) using Gmail API
func (s *FetcherService) getGmailFolders(account models.EmailAccount) ([]string, error) {
	s.logger.Debug("Getting Gmail folders using Gmail API for %s", account.EmailAddress)

	// 使用Gmail API获取标签
	mailboxes, err := s.getGmailMailboxes(account)
	if err != nil {
		s.logger.Error("Failed to get Gmail mailboxes: %v", err)
		return nil, fmt.Errorf("failed to get Gmail mailboxes: %w", err)
	}

	// 转换为文件夹名称列表
	var folders []string
	for _, mailbox := range mailboxes {
		folders = append(folders, mailbox.Name)
	}

	s.logger.Info("Retrieved %d Gmail labels for %s: %v", len(folders), account.EmailAddress, folders)
	return folders, nil
}

// getImapFolders retrieves IMAP folders using IMAP LIST command
func (s *FetcherService) getImapFolders(account models.EmailAccount) ([]string, error) {
	s.logger.Debug("Getting IMAP folders for %s using real IMAP connection", account.EmailAddress)

	// 使用真正的IMAP连接获取文件夹列表
	c, err := s.connectAndAuthenticateIMAP(account)
	if err != nil {
		s.logger.Error("Failed to connect to IMAP server: %v", err)
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Logout()

	// 使用LIST命令获取所有文件夹
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var folders []string
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}

	if err := <-done; err != nil {
		s.logger.Error("IMAP LIST command failed: %v", err)
		return nil, fmt.Errorf("IMAP LIST command failed: %w", err)
	}

	s.logger.Info("Retrieved %d folders from IMAP server for %s: %v", len(folders), account.EmailAddress, folders)
	return folders, nil
}

// connectAndAuthenticateIMAP connects to IMAP server and authenticates
func (s *FetcherService) connectAndAuthenticateIMAP(account models.EmailAccount) (*client.Client, error) {
	var c *client.Client
	var err error

	// Check if MailProvider is nil
	if account.MailProvider == nil {
		return nil, fmt.Errorf("mail provider is not configured for account %s", account.EmailAddress)
	}

	serverAddr := fmt.Sprintf("%s:%d", account.MailProvider.IMAPServer, account.MailProvider.IMAPPort)
	s.logger.Info("Connecting to IMAP server %s for %s", serverAddr, account.EmailAddress)

	if account.Proxy != "" {
		proxyURL, err := url.Parse(account.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		dialer, err := s.createProxyDialer(proxyURL)
		if err != nil {
			s.logger.Error("Failed to create proxy dialer: %v", err)
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}

		s.logger.Debug("Connecting via %s proxy: %s", proxyURL.Scheme, account.Proxy)

		// For IMAP over proxy, we need to handle TLS after CONNECT
		if account.MailProvider.IMAPPort == 993 {
			// First establish the proxy tunnel
			proxyConn, err := dialer.Dial("tcp", serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}

			// Then wrap with TLS
			s.logger.Debug("Establishing TLS connection through proxy tunnel")
			tlsConn := tls.Client(proxyConn, &tls.Config{
				ServerName: account.MailProvider.IMAPServer,
			})

			// Perform TLS handshake
			if err := tlsConn.Handshake(); err != nil {
				proxyConn.Close()
				s.logger.Error("TLS handshake failed: %v", err)
				return nil, fmt.Errorf("TLS handshake failed: %w", err)
			}

			// Create IMAP client with the TLS connection
			c, err = client.New(tlsConn)
			if err != nil {
				tlsConn.Close()
				s.logger.Error("Failed to create IMAP client: %v", err)
				return nil, fmt.Errorf("failed to create IMAP client: %w", err)
			}
		} else {
			// For non-TLS IMAP, use the proxy connection directly
			c, err = client.DialWithDialer(dialer, serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial via proxy: %v", err)
				return nil, fmt.Errorf("failed to dial via proxy: %w", err)
			}
		}
	} else {
		// Use TLS connection for secure IMAP (port 993)
		if account.MailProvider.IMAPPort == 993 {
			s.logger.Debug("Using TLS connection for port 993")
			c, err = client.DialTLS(serverAddr, &tls.Config{ServerName: account.MailProvider.IMAPServer})
			if err != nil {
				s.logger.Error("Failed to dial with TLS: %v", err)
				return nil, fmt.Errorf("failed to dial with TLS: %w", err)
			}
		} else {
			// Use plain connection for non-secure IMAP (port 143)
			s.logger.Debug("Using plain connection for port %d", account.MailProvider.IMAPPort)
			c, err = client.Dial(serverAddr)
			if err != nil {
				s.logger.Error("Failed to dial: %v", err)
				return nil, fmt.Errorf("failed to dial: %w", err)
			}
		}
	}

	// Login based on auth type
	s.logger.Debug("Authenticating with auth type: %s", account.AuthType)
	switch account.AuthType {
	case models.AuthTypePassword:
		// Standard password authentication
		if err := c.Login(account.EmailAddress, account.Password); err != nil {
			s.logger.Error("Password authentication failed for %s: %v", account.EmailAddress, err)
			c.Logout()
			return nil, fmt.Errorf("login failed: %w", err)
		}
	case models.AuthTypeOAuth2:
		// OAuth2 authentication
		s.logger.Debug("Using OAuth2 authentication")
		// Get client_id from CustomSettings, with fallback to global config
		clientID, ok := account.CustomSettings["client_id"]
		if !ok {
			s.logger.Warn("client_id not found in custom settings, trying to get from global config")

			// Try to get client_id from global OAuth2 config
			oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())
			var tempConfig *models.OAuth2GlobalConfig
			var err error

			// First try by OAuth2ProviderID if available
			if account.OAuth2ProviderID != nil {
				tempConfig, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
				if err != nil {
					s.logger.Warn("Failed to get config by OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
				}
			}

			// Fallback to provider type
			if tempConfig == nil {
				tempConfig, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
				if err != nil {
					s.logger.Error("Failed to get global config for provider %s: %v", account.MailProvider.Type, err)
					c.Logout()
					return nil, fmt.Errorf("client_id not found in custom settings and failed to get from global config: %w", err)
				}
			}

			if tempConfig == nil {
				c.Logout()
				return nil, fmt.Errorf("client_id not found in custom settings and no global config available for provider %s", account.MailProvider.Type)
			}

			clientID = tempConfig.ClientID
			s.logger.Info("Using client_id from global config (ID: %d, Name: %s) for connectAndAuthenticateIMAP", tempConfig.ID, tempConfig.Name)
		}

		refreshToken, ok := account.CustomSettings["refresh_token"]
		if !ok {
			s.logger.Error("refresh_token not found in custom settings")
			c.Logout()
			return nil, fmt.Errorf("refresh_token not found in custom settings")
		}

		// Get client_secret from global OAuth2 config (secure approach)
		clientSecret := ""
		oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

		var config *models.OAuth2GlobalConfig
		var err error

		// Priority 1: Use OAuth2ProviderID if available (new multi-config support)
		if account.OAuth2ProviderID != nil && *account.OAuth2ProviderID > 0 {
			s.logger.Debug("Using OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from OAuth2ProviderID %d for account %s", *account.OAuth2ProviderID, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get config from OAuth2ProviderID %d for account %s: %v", *account.OAuth2ProviderID, account.EmailAddress, err)
			}
		}

		// Priority 2: Fallback to provider type lookup (backward compatibility)
		if config == nil {
			s.logger.Debug("Falling back to provider type lookup for %s", account.MailProvider.Type)
			config, err = oauth2GlobalConfigRepo.GetByProviderType(account.MailProvider.Type)
			if err == nil && config != nil {
				clientSecret = config.ClientSecret
				s.logger.Debug("Retrieved client_secret from provider type %s for account %s", account.MailProvider.Type, account.EmailAddress)
			} else {
				s.logger.Warn("Failed to get client_secret from provider type %s for account %s: %v", account.MailProvider.Type, account.EmailAddress, err)
			}
		}

		// Refresh access token - use cached method with concurrency protection for better reliability
		s.logger.Debug("Refreshing OAuth2 access token for IMAP folder listing with cache")
		accessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
			string(account.MailProvider.Type),
			clientID,
			clientSecret,
			refreshToken,
			account.ID,
			account.Proxy, // Pass proxy settings if available
		)
		if err != nil {
			s.logger.Error("Failed to refresh access token: %v", err)
			c.Logout()
			return nil, fmt.Errorf("failed to refresh access token: %w", err)
		}

		// Update access token in account
		if account.CustomSettings == nil {
			account.CustomSettings = make(models.JSONMap)
		}
		// 并发安全的CustomSettings更新
		newCustomSettings := make(models.JSONMap)
		if account.CustomSettings != nil {
			for k, v := range account.CustomSettings {
				newCustomSettings[k] = v
			}
		}
		newCustomSettings["access_token"] = accessToken
		account.CustomSettings = newCustomSettings

		// Update the account with new access token
		updatedAccount := account
		if err := s.accountRepo.Update(&updatedAccount); err != nil {
			s.logger.Warn("Failed to update access token in database: %v", err)
		}

		// Authenticate with OAuth2
		saslClient := NewOAuth2SASLClient(account.EmailAddress, accessToken)
		if err := c.Authenticate(saslClient); err != nil {
			s.logger.Error("OAuth2 authentication failed: %v", err)
			c.Logout()
			return nil, fmt.Errorf("OAuth2 authentication failed: %w", err)
		}

		// Check connection state after authentication
		if c.State() != imap.AuthenticatedState && c.State() != imap.SelectedState {
			s.logger.Error("IMAP connection is in unexpected state after OAuth2 authentication: %s", c.State())
			c.Logout()
			return nil, fmt.Errorf("IMAP connection is in unexpected state after OAuth2 authentication: %s", c.State())
		}

		s.logger.Debug("IMAP connection state after OAuth2 authentication: %s", c.State())

		// For Microsoft Outlook, sometimes we need to send a NOOP command to refresh the connection
		if account.MailProvider.Type == models.ProviderTypeOutlook {
			s.logger.Debug("Sending NOOP command to refresh Outlook connection state")
			if err := c.Noop(); err != nil {
				s.logger.Warn("NOOP command failed, but continuing: %v", err)
			} else {
				s.logger.Debug("NOOP command successful, connection state: %s", c.State())
			}
		}
	default:
		s.logger.Error("Unsupported auth type: %s", account.AuthType)
		c.Logout()
		return nil, fmt.Errorf("unsupported auth type: %s", account.AuthType)
	}

	s.logger.Info("Successfully connected and logged in for %s using %s auth", account.EmailAddress, account.AuthType)
	return c, nil
}

// shouldUseGmailAPI determines if should use Gmail API instead of IMAP
func (s *FetcherService) shouldUseGmailAPI(account models.EmailAccount) bool {
	return account.AuthType == models.AuthTypeOAuth2 &&
		account.MailProvider != nil &&
		account.MailProvider.Type == models.ProviderTypeGmail
}

// fetchEmailsFromGmailAPI fetches emails using Gmail API
func (s *FetcherService) fetchEmailsFromGmailAPI(account models.EmailAccount, options FetchEmailsOptions) ([]models.Email, error) {
	s.logger.Debug("Fetching emails using Gmail API for account %s", account.EmailAddress)

	// Create Gmail API service
	gmailService, err := s.createGmailService(account)
	if err != nil {
		s.logger.Error("Failed to create Gmail service: %v", err)
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Get sync config to check for History ID
	syncConfigRepo := repository.NewSyncConfigRepository(s.accountRepo.GetDB())
	syncConfig, err := syncConfigRepo.GetByAccountID(account.ID)
	if err != nil {
		s.logger.Warn("Failed to get sync config for account %d: %v", account.ID, err)
		// Continue with full sync if no config found
	}

	var emails []models.Email
	var newHistoryID string

	// Try incremental sync using History API if we have a previous History ID
	if syncConfig != nil && syncConfig.LastHistoryID != "" {
		s.logger.Debug("Attempting Gmail unified incremental sync using History ID: %s", syncConfig.LastHistoryID)

		// Use unified Gmail API sync - gets ALL email changes in one call
		historyEmails, historyID, err := s.fetchGmailHistoryChangesUnified(gmailService, syncConfig.LastHistoryID, account.ID, options)
		if err != nil {
			s.logger.Warn("Gmail unified History API sync failed, falling back to full sync: %v", err)
			// Fall back to full sync
			messages, err := s.fetchGmailMessagesUnified(gmailService, options)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch Gmail messages (unified): %w", err)
			}
			emails = s.convertGmailMessages(messages, account.ID)
		} else {
			emails = historyEmails
			newHistoryID = historyID
		}
	} else {
		s.logger.Debug("No previous History ID found, performing Gmail unified full sync")
		// Full sync for first time or when no history ID available
		messages, err := s.fetchGmailMessagesUnified(gmailService, options)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Gmail messages (unified): %w", err)
		}
		emails = s.convertGmailMessages(messages, account.ID)
	}

	// Get current profile to update History ID
	if newHistoryID == "" {
		profile, err := gmailService.Users.GetProfile("me").Do()
		if err != nil {
			s.logger.Warn("Failed to get user profile for History ID: %v", err)
		} else {
			newHistoryID = fmt.Sprintf("%d", profile.HistoryId)
		}
	}

	// Update sync config with new History ID
	if syncConfig != nil && newHistoryID != "" {
		syncConfig.LastHistoryID = newHistoryID
		if err := syncConfigRepo.Update(syncConfig); err != nil {
			s.logger.Warn("Failed to update History ID in sync config: %v", err)
		} else {
			s.logger.Debug("Updated History ID to: %s", newHistoryID)
		}
	}

	// Update last sync time
	if err := s.accountRepo.UpdateLastSync(account.ID); err != nil {
		s.logger.Warn("Failed to update last sync time: %v", err)
	}

	s.logger.Debug("email: %s, historyId: %s, newEmails: %d", account.EmailAddress, newHistoryID, len(emails))
	return emails, nil
}

// createGmailService creates a Gmail API service client
func (s *FetcherService) createGmailService(account models.EmailAccount) (*gmail.Service, error) {
	// Get OAuth2 configuration
	oauth2GlobalConfigRepo := repository.NewOAuth2GlobalConfigRepository(s.accountRepo.GetDB())

	var oauth2Config *models.OAuth2GlobalConfig
	var err error

	// Priority 1: Use OAuth2ProviderID if available
	if account.OAuth2ProviderID != nil && *account.OAuth2ProviderID > 0 {
		s.logger.Debug("Using OAuth2ProviderID %d for Gmail service", *account.OAuth2ProviderID)
		oauth2Config, err = oauth2GlobalConfigRepo.GetByID(*account.OAuth2ProviderID)
		if err != nil {
			s.logger.Warn("Failed to get config from OAuth2ProviderID %d: %v", *account.OAuth2ProviderID, err)
		}
	}

	// Priority 2: Fallback to provider type lookup
	if oauth2Config == nil {
		s.logger.Debug("Falling back to provider type lookup for Gmail")
		oauth2Config, err = oauth2GlobalConfigRepo.GetByProviderType(models.ProviderTypeGmail)
		if err != nil {
			return nil, fmt.Errorf("failed to get OAuth2 config: %w", err)
		}
	}

	if oauth2Config == nil {
		return nil, fmt.Errorf("no OAuth2 config found for Gmail")
	}

	// Get tokens from CustomSettings
	if account.CustomSettings == nil {
		return nil, fmt.Errorf("OAuth2 tokens not found in account settings")
	}

	accessToken, ok := account.CustomSettings["access_token"]
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("access_token not found in account settings")
	}

	refreshToken, ok := account.CustomSettings["refresh_token"]
	if !ok || refreshToken == "" {
		return nil, fmt.Errorf("refresh_token not found in account settings")
	}

	// Parse token expiry
	var tokenExpiry time.Time
	if expiryStr, exists := account.CustomSettings["expires_at"]; exists && expiryStr != "" {
		if expiryInt, err := strconv.ParseInt(expiryStr, 10, 64); err == nil {
			tokenExpiry = time.Unix(expiryInt, 0)
		} else if expiryTime, err := time.Parse(time.RFC3339, expiryStr); err == nil {
			tokenExpiry = expiryTime
		}
	}

	// Check if token is expired and refresh if necessary (使用带缓存和并发控制的方法)
	if tokenExpiry.IsZero() || time.Now().After(tokenExpiry.Add(-5*time.Minute)) {
		s.logger.Debug("Access token is expired or about to expire, refreshing token for Gmail API")

		// 使用带缓存和并发控制的token刷新方法，并传递代理配置
		newAccessToken, err := s.oauth2Service.RefreshAccessTokenWithCacheAndProxy(
			string(models.ProviderTypeGmail),
			oauth2Config.ClientID,
			oauth2Config.ClientSecret,
			refreshToken,
			account.ID,
			account.Proxy, // 传递代理配置
		)
		if err != nil {
			s.logger.Error("Failed to refresh access token for Gmail API: %v", err)
			return nil, fmt.Errorf("failed to refresh access token: %w", err)
		}

		// Update access token in account - 并发安全更新
		newCustomSettings := make(models.JSONMap)
		if account.CustomSettings != nil {
			for k, v := range account.CustomSettings {
				newCustomSettings[k] = v
			}
		}
		newCustomSettings["access_token"] = newAccessToken
		newCustomSettings["expires_at"] = fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix())
		account.CustomSettings = newCustomSettings

		// Update the account with new access token
		if err := s.accountRepo.Update(&account); err != nil {
			s.logger.Warn("Failed to update access token in database: %v", err)
		} else {
			s.logger.Debug("Successfully updated access token in database")
		}

		// Use new access token
		accessToken = newAccessToken
		tokenExpiry = time.Now().Add(time.Hour)
	}

	// Create OAuth2 config
	config := &oauth2.Config{
		ClientID:     oauth2Config.ClientID,
		ClientSecret: oauth2Config.ClientSecret,
		Scopes:       oauth2Config.Scopes,
		Endpoint:     google.Endpoint,
	}

	// Create token
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       tokenExpiry,
		TokenType:    "Bearer",
	}

	// Create HTTP client with OAuth2 and proxy support
	ctx := context.Background()

	// Create base HTTP client with proxy support if configured
	var baseClient *http.Client
	if account.Proxy != "" {
		s.logger.Debug("Creating HTTP client with proxy for Gmail API: %s", account.Proxy)
		baseClient, err = s.createHTTPClientWithProxy(account.Proxy)
		if err != nil {
			s.logger.Error("Failed to create HTTP client with proxy: %v", err)
			return nil, fmt.Errorf("failed to create HTTP client with proxy: %w", err)
		}
	} else {
		s.logger.Debug("Creating HTTP client without proxy for Gmail API")
		baseClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create OAuth2 client using the base client
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: config.TokenSource(ctx, token),
			Base:   baseClient.Transport,
		},
		Timeout: baseClient.Timeout,
	}

	// Create Gmail service
	service, err := gmail.NewService(ctx, option.WithHTTPClient(oauth2Client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}

// fetchGmailMessages fetches messages from Gmail API
func (s *FetcherService) fetchGmailMessages(service *gmail.Service, options FetchEmailsOptions) ([]*gmail.Message, error) {
	// Build query based on options
	query := s.buildGmailQuery(service, options)

	// Set mailbox/label filter by dynamically getting Gmail label ID
	labelIDs := []string{}
	if options.Mailbox != "" && options.Mailbox != "INBOX" {
		// Use dynamic label ID lookup
		labelID, err := s.getGmailLabelID(service, options.Mailbox)
		if err != nil {
			s.logger.Warn("Failed to get Gmail label ID for mailbox '%s': %v", options.Mailbox, err)
			// Fall back to searching all messages if label not found
			s.logger.Debug("Falling back to search all messages without label filter")
		} else {
			labelIDs = append(labelIDs, labelID)
		}
	} else {
		labelIDs = append(labelIDs, "INBOX")
	}

	// List messages
	listCall := service.Users.Messages.List("me").Q(query)
	if len(labelIDs) > 0 {
		listCall = listCall.LabelIds(labelIDs...)
	}

	// Set limit
	limit := int64(options.Limit)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	listCall = listCall.MaxResults(limit)

	s.logger.Debug("Fetching Gmail messages with query: %s, labels: %v, limit: %d", query, labelIDs, limit)

	listResp, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list Gmail messages: %w", err)
	}

	if len(listResp.Messages) == 0 {
		s.logger.Debug("No messages found for the query")
		return []*gmail.Message{}, nil
	}

	// Fetch full message details
	var messages []*gmail.Message
	for _, msgRef := range listResp.Messages {
		msg, err := service.Users.Messages.Get("me", msgRef.Id).Do()
		if err != nil {
			s.logger.Warn("Failed to get message %s: %v", msgRef.Id, err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// buildGmailQuery builds Gmail search query based on options
func (s *FetcherService) buildGmailQuery(service *gmail.Service, options FetchEmailsOptions) string {
	var queryParts []string

	// Add mailbox filter using Gmail search syntax
	if options.Mailbox != "" {
		// For Gmail API, we need to use the correct search syntax
		// Don't add label filters here as they cause issues with custom labels
		// Instead, we'll handle this in the message listing
		s.logger.Debug("Mailbox filter '%s' will be handled during message listing", options.Mailbox)
	}

	// Date filter
	if options.StartDate != nil {
		queryParts = append(queryParts, fmt.Sprintf("after:%s", options.StartDate.Format("2006/01/02")))
	}
	if options.EndDate != nil {
		queryParts = append(queryParts, fmt.Sprintf("before:%s", options.EndDate.Format("2006/01/02")))
	}

	// Search query
	if options.SearchQuery != "" {
		queryParts = append(queryParts, options.SearchQuery)
	}

	query := strings.Join(queryParts, " ")
	if query == "" {
		query = "in:inbox" // Default query
	}

	return query
}

// convertGmailMessage converts Gmail message to Email model
func (s *FetcherService) convertGmailMessage(gmailMsg *gmail.Message, accountID uint) (*models.Email, error) {
	email := &models.Email{
		MessageID: gmailMsg.Id, // Use Gmail message ID
		AccountID: accountID,
		Size:      int64(gmailMsg.SizeEstimate),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Parse headers
	for _, header := range gmailMsg.Payload.Headers {
		switch header.Name {
		case "Message-ID":
			// Store original RFC Message-ID if available
			if header.Value != "" {
				email.MessageID = header.Value
			}
		case "Subject":
			email.Subject = header.Value
		case "From":
			email.From = models.StringSlice{header.Value}
		case "To":
			email.To = models.StringSlice{header.Value}
		case "Cc":
			email.Cc = models.StringSlice{header.Value}
		case "Bcc":
			email.Bcc = models.StringSlice{header.Value}
		case "Date":
			// Try multiple date formats to handle Gmail's various date formats
			dateFormats := []string{
				time.RFC1123Z,                          // "Mon, 02 Jan 2006 15:04:05 -0700"
				time.RFC1123,                           // "Mon, 02 Jan 2006 15:04:05 MST"
				"Mon, 2 Jan 2006 15:04:05 -0700",       // Gmail format without leading zero
				"Mon, 2 Jan 2006 15:04:05 MST",         // Gmail format without leading zero (MST)
				"Mon, 2 Jan 2006 15:04:05 +0000",       // Gmail format with +0000 timezone
				"Mon, 02 Jan 2006 15:04:05 +0000",      // Gmail format with leading zero and +0000
				"Mon, 2 Jan 2006 15:04:05 GMT",         // Gmail format with GMT
				"Mon, 02 Jan 2006 15:04:05 GMT",        // Gmail format with leading zero and GMT
				"2 Jan 2006 15:04:05 -0700",            // Without weekday
				"2 Jan 2006 15:04:05 +0000",            // Without weekday, +0000 timezone
				"02 Jan 2006 15:04:05 +0000",           // With leading zero, no weekday, +0000
				"2006-01-02 15:04:05 -0700",            // ISO-like format
				"2006-01-02 15:04:05 +0000",            // ISO-like format with +0000
				time.RFC3339,                           // "2006-01-02T15:04:05Z07:00"
				time.RFC822Z,                           // "02 Jan 06 15:04 -0700"
				time.RFC822,                            // "02 Jan 06 15:04 MST"
				"Mon, 2 Jan 2006 15:04:05 -0700 (MST)", // With timezone name in parentheses
				"Mon, 2 Jan 2006 15:04:05 +0000 (UTC)", // With UTC in parentheses
			}

			var parsedSuccessfully bool
			for i, format := range dateFormats {
				if parsedDate, err := time.Parse(format, header.Value); err == nil {
					email.Date = parsedDate
					email.ReceivedAt = parsedDate // Also set ReceivedAt
					parsedSuccessfully = true
					s.logger.Debug("Successfully parsed date '%s' using format %d (%s) for message %s",
						header.Value, i, format, gmailMsg.Id)
					break
				}
			}

			// Enhanced logging if date parsing fails
			if !parsedSuccessfully {
				s.logger.Error("Failed to parse date '%s' for message %s. Tried %d formats. Setting to zero time.",
					header.Value, gmailMsg.Id, len(dateFormats))
				// Set to zero time explicitly
				email.Date = time.Time{}
				email.ReceivedAt = time.Time{}
			}
		}
	}

	// Handle Gmail labels - store all labels in Flags, primary label in MailboxName
	if len(gmailMsg.LabelIds) > 0 {
		email.Flags = models.StringSlice(gmailMsg.LabelIds)
		email.MailboxName = s.getPrimaryMailboxFromLabels(gmailMsg.LabelIds)
	} else {
		email.MailboxName = "INBOX"
	}

	// Extract body content
	if gmailMsg.Payload != nil {
		email.Body, email.HTMLBody = s.extractGmailBody(gmailMsg.Payload)
	}

	// Use snippet if no body extracted
	if email.Body == "" && gmailMsg.Snippet != "" {
		email.Body = gmailMsg.Snippet
	}

	return email, nil
}

// getPrimaryMailboxFromLabels determines primary mailbox from Gmail labels
func (s *FetcherService) getPrimaryMailboxFromLabels(labels []string) string {
	// Priority mapping
	priority := map[string]int{
		"INBOX":     1,
		"SENT":      2,
		"DRAFT":     3,
		"SPAM":      4,
		"TRASH":     5,
		"IMPORTANT": 6,
		"STARRED":   7,
	}

	bestLabel := "INBOX" // Default
	bestPriority := 999

	for _, label := range labels {
		if p, exists := priority[label]; exists && p < bestPriority {
			bestLabel = label
			bestPriority = p
		}
	}

	return bestLabel
}

// extractGmailBody extracts text and HTML body from Gmail message payload
func (s *FetcherService) extractGmailBody(payload *gmail.MessagePart) (string, string) {
	var textBody, htmlBody string

	// Check if this part has body data
	if payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			content := string(decoded)

			// Determine content type
			mimeType := "text/plain"
			for _, header := range payload.Headers {
				if header.Name == "Content-Type" {
					mimeType = header.Value
					break
				}
			}

			if strings.Contains(mimeType, "text/html") {
				htmlBody = content
			} else {
				textBody = content
			}
		}
	}

	// Recursively check parts
	for _, part := range payload.Parts {
		partText, partHTML := s.extractGmailBody(part)
		if partText != "" {
			textBody = partText
		}
		if partHTML != "" {
			htmlBody = partHTML
		}
	}

	return textBody, htmlBody
}

// convertGmailMessages batch converts Gmail messages to Email models
func (s *FetcherService) convertGmailMessages(messages []*gmail.Message, accountID uint) []models.Email {
	var emails []models.Email
	for _, msg := range messages {
		email, err := s.convertGmailMessage(msg, accountID)
		if err != nil {
			s.logger.Warn("Failed to convert Gmail message %s: %v", msg.Id, err)
			continue
		}
		emails = append(emails, *email)
	}
	return emails
}

// fetchGmailHistoryChanges fetches email changes using Gmail History API
func (s *FetcherService) fetchGmailHistoryChanges(service *gmail.Service, startHistoryID string, accountID uint, options FetchEmailsOptions) ([]models.Email, string, error) {
	s.logger.Debug("Fetching Gmail history changes from History ID: %s", startHistoryID)

	// Parse start history ID
	historyID, err := strconv.ParseUint(startHistoryID, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("invalid history ID: %w", err)
	}

	// Get Gmail label ID for the specified mailbox for later filtering
	var targetLabelID string
	if options.Mailbox != "" && options.Mailbox != "INBOX" {
		targetLabelID, err = s.getGmailLabelID(service, options.Mailbox)
		if err != nil {
			s.logger.Warn("Failed to get Gmail label ID for mailbox '%s': %v", options.Mailbox, err)
			// Continue without label filter
		}
	} else {
		targetLabelID = "INBOX"
	}

	// Call History API WITHOUT label filter to get all changes
	// This ensures we don't miss new emails that haven't been labeled yet
	historyCall := service.Users.History.List("me").StartHistoryId(historyID)
	// DO NOT set LabelId filter here - we want all changes

	s.logger.Debug("Calling History API without label filter to capture all changes")
	historyResp, err := historyCall.Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get history: %w", err)
	}

	if len(historyResp.History) == 0 {
		s.logger.Debug("No history changes found")
		return []models.Email{}, fmt.Sprintf("%d", historyResp.HistoryId), nil
	}

	// Collect unique message IDs from history changes
	messageIDSet := make(map[string]bool)
	for _, history := range historyResp.History {
		// Messages added
		for _, msgAdded := range history.MessagesAdded {
			if msgAdded.Message != nil {
				messageIDSet[msgAdded.Message.Id] = true
			}
		}
		// Messages deleted - we could handle this differently if needed
		for _, msgDeleted := range history.MessagesDeleted {
			if msgDeleted.Message != nil {
				// For now, we'll still fetch it to mark as deleted in our system
				messageIDSet[msgDeleted.Message.Id] = true
			}
		}
		// Label changes
		for _, labelAdded := range history.LabelsAdded {
			if labelAdded.Message != nil {
				messageIDSet[labelAdded.Message.Id] = true
			}
		}
		for _, labelRemoved := range history.LabelsRemoved {
			if labelRemoved.Message != nil {
				messageIDSet[labelRemoved.Message.Id] = true
			}
		}
	}

	s.logger.Debug("Found %d unique message IDs in history changes", len(messageIDSet))

	// Fetch full message details for changed messages
	var messages []*gmail.Message
	for messageID := range messageIDSet {
		msg, err := service.Users.Messages.Get("me", messageID).Do()
		if err != nil {
			s.logger.Warn("Failed to get message %s: %v", messageID, err)
			continue
		}
		messages = append(messages, msg)
	}

	// Filter messages by target label if specified
	var filteredByLabel []*gmail.Message
	if targetLabelID != "" {
		for _, msg := range messages {
			// Check if message has the target label
			hasTargetLabel := false
			for _, labelID := range msg.LabelIds {
				if labelID == targetLabelID {
					hasTargetLabel = true
					break
				}
			}
			if hasTargetLabel {
				filteredByLabel = append(filteredByLabel, msg)
			}
		}
		s.logger.Debug("Filtered %d messages by label '%s' (target: %s)", len(filteredByLabel), options.Mailbox, targetLabelID)
	} else {
		filteredByLabel = messages
	}

	// Apply additional options filters (date, search query, etc.)
	filteredMessages := s.filterGmailMessages(filteredByLabel, options)

	s.logger.Info("Found %d changed messages in history, %d after label filtering, %d after all filters", len(messages), len(filteredByLabel), len(filteredMessages))

	// Convert to Email models
	emails := s.convertGmailMessages(filteredMessages, accountID)

	return emails, fmt.Sprintf("%d", historyResp.HistoryId), nil
}

// filterGmailMessages applies filtering options to Gmail messages
func (s *FetcherService) filterGmailMessages(messages []*gmail.Message, options FetchEmailsOptions) []*gmail.Message {
	var filtered []*gmail.Message

	for _, msg := range messages {
		// Apply date filter
		if options.StartDate != nil || options.EndDate != nil {
			msgDate := s.getGmailMessageDate(msg)
			if msgDate.IsZero() {
				continue // Skip messages without valid date
			}

			if options.StartDate != nil && msgDate.Before(*options.StartDate) {
				continue
			}
			if options.EndDate != nil && msgDate.After(*options.EndDate) {
				continue
			}
		}

		// Apply search query filter (basic implementation)
		if options.SearchQuery != "" {
			if !s.messageMatchesQuery(msg, options.SearchQuery) {
				continue
			}
		}

		filtered = append(filtered, msg)

		// Apply limit
		if options.Limit > 0 && len(filtered) >= options.Limit {
			break
		}
	}

	return filtered
}

// getGmailMessageDate extracts date from Gmail message headers
func (s *FetcherService) getGmailMessageDate(msg *gmail.Message) time.Time {
	if msg.Payload == nil {
		return time.Time{}
	}

	for _, header := range msg.Payload.Headers {
		if header.Name == "Date" {
			// Try multiple date formats to handle Gmail's various date formats
			dateFormats := []string{
				time.RFC1123Z,                    // "Mon, 02 Jan 2006 15:04:05 -0700"
				time.RFC1123,                     // "Mon, 02 Jan 2006 15:04:05 MST"
				"Mon, 2 Jan 2006 15:04:05 -0700", // Gmail format without leading zero
				"Mon, 2 Jan 2006 15:04:05 MST",   // Gmail format without leading zero (MST)
				"2 Jan 2006 15:04:05 -0700",      // Without weekday
				"2006-01-02 15:04:05 -0700",      // ISO-like format
				time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
			}

			for _, format := range dateFormats {
				if parsedDate, err := time.Parse(format, header.Value); err == nil {
					return parsedDate
				}
			}

			// Log if date parsing fails
			s.logger.Warn("Failed to parse date '%s' for message %s", header.Value, msg.Id)
		}
	}
	return time.Time{}
}

// messageMatchesQuery checks if a Gmail message matches the search query
func (s *FetcherService) messageMatchesQuery(msg *gmail.Message, query string) bool {
	query = strings.ToLower(query)

	// Check snippet
	if strings.Contains(strings.ToLower(msg.Snippet), query) {
		return true
	}

	// Check headers
	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			headerValue := strings.ToLower(header.Value)
			if strings.Contains(headerValue, query) {
				return true
			}
		}
	}

	return false
}

// getGmailLabelID dynamically gets the Gmail label ID for a given mailbox name
func (s *FetcherService) getGmailLabelID(service *gmail.Service, mailboxName string) (string, error) {
	// Get all labels from Gmail
	labelList, err := service.Users.Labels.List("me").Do()
	if err != nil {
		return "", fmt.Errorf("failed to get labels: %w", err)
	}

	// First, try to match by exact name
	for _, label := range labelList.Labels {
		if label.Name == mailboxName {
			return label.Id, nil
		}
	}

	// If no exact match, try to match common mappings
	// Map IMAP folder names to Gmail system labels
	systemLabelMap := map[string]string{
		"INBOX":     "INBOX",
		"SENT":      "SENT",
		"DRAFTS":    "DRAFT",
		"TRASH":     "TRASH",
		"SPAM":      "SPAM",
		"IMPORTANT": "IMPORTANT",
		"STARRED":   "STARRED",
	}

	// Check if it's a system label
	if labelID, exists := systemLabelMap[strings.ToUpper(mailboxName)]; exists {
		return labelID, nil
	}

	// Try to find by matching common Gmail folder patterns
	for _, label := range labelList.Labels {
		// Match patterns like [Gmail]/已发邮件 with SENT label
		if strings.Contains(mailboxName, "已发邮件") || strings.Contains(mailboxName, "Sent Mail") {
			if label.Id == "SENT" {
				return label.Id, nil
			}
		}
		if strings.Contains(mailboxName, "已加星标") || strings.Contains(mailboxName, "Starred") {
			if label.Id == "STARRED" {
				return label.Id, nil
			}
		}
	}

	return "", fmt.Errorf("label not found: %s", mailboxName)
}

// fetchGmailHistoryChangesUnified fetches ALL email changes using Gmail History API without label filtering
func (s *FetcherService) fetchGmailHistoryChangesUnified(service *gmail.Service, startHistoryID string, accountID uint, options FetchEmailsOptions) ([]models.Email, string, error) {
	s.logger.Debug("=== Gmail History API Debug ===")
	s.logger.Debug("Starting History ID: %s", startHistoryID)

	// Parse start history ID
	historyID, err := strconv.ParseUint(startHistoryID, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("invalid history ID: %w", err)
	}

	// Call History API WITH pagination handling to get ALL changes
	s.logger.Debug("Calling Gmail History API with startHistoryId=%d", historyID)

	// Collect unique message IDs from ALL history changes across all pages
	messageIDSet := make(map[string]bool)
	var finalHistoryID uint64
	pageToken := ""
	pageCount := 0

	for {
		historyCall := service.Users.History.List("me").StartHistoryId(historyID)
		if pageToken != "" {
			historyCall = historyCall.PageToken(pageToken)
		}

		historyResp, err := historyCall.Do()
		if err != nil {
			s.logger.Error("Gmail History API call failed on page %d: %v", pageCount+1, err)
			return nil, "", fmt.Errorf("failed to get history: %w", err)
		}

		pageCount++
		finalHistoryID = historyResp.HistoryId

		s.logger.Debug("Gmail History API Response (page %d):", pageCount)
		s.logger.Debug("  - Current History ID: %d", historyResp.HistoryId)
		s.logger.Debug("  - History entries count: %d", len(historyResp.History))
		s.logger.Debug("  - Next Page Token: %s", historyResp.NextPageToken)

		// Process current page's history changes
		for _, history := range historyResp.History {
			// Messages added
			for _, msgAdded := range history.MessagesAdded {
				if msgAdded.Message != nil {
					messageIDSet[msgAdded.Message.Id] = true
				}
			}
			// Messages deleted
			for _, msgDeleted := range history.MessagesDeleted {
				if msgDeleted.Message != nil {
					messageIDSet[msgDeleted.Message.Id] = true
				}
			}
			// Label changes
			for _, labelAdded := range history.LabelsAdded {
				if labelAdded.Message != nil {
					messageIDSet[labelAdded.Message.Id] = true
				}
			}
			for _, labelRemoved := range history.LabelsRemoved {
				if labelRemoved.Message != nil {
					messageIDSet[labelRemoved.Message.Id] = true
				}
			}
		}

		// Check if there are more pages
		if historyResp.NextPageToken == "" {
			break
		}
		pageToken = historyResp.NextPageToken

		// Safety check to prevent infinite loops
		if pageCount >= 100 {
			s.logger.Warn("Reached maximum page limit (100) for History API, stopping pagination")
			break
		}
	}

	s.logger.Debug("Processed %d pages from Gmail History API", pageCount)

	if len(messageIDSet) == 0 {
		return []models.Email{}, fmt.Sprintf("%d", finalHistoryID), nil
	}

	s.logger.Debug("Found %d unique message IDs in unified history changes", len(messageIDSet))

	// Fetch full message details for all changed messages
	var messages []*gmail.Message
	for messageID := range messageIDSet {
		msg, err := service.Users.Messages.Get("me", messageID).Do()
		if err != nil {
			s.logger.Warn("Failed to get message %s: %v", messageID, err)
			continue
		}
		messages = append(messages, msg)
	}

	// For incremental sync via History API, we don't need date filtering
	// History API already provides incremental changes since last sync
	s.logger.Debug("Gmail unified incremental sync: found %d changed messages", len(messages))

	// Convert to Email models directly - Gmail labels are stored in LabelIds
	emails := s.convertGmailMessages(messages, accountID)

	return emails, fmt.Sprintf("%d", finalHistoryID), nil
}

// fetchGmailMessagesUnified fetches Gmail messages for full sync without label filtering with pagination
func (s *FetcherService) fetchGmailMessagesUnified(service *gmail.Service, options FetchEmailsOptions) ([]*gmail.Message, error) {
	s.logger.Debug("Fetching Gmail messages (unified full sync with pagination)")

	// Build query based on options (date, search) but NOT mailbox/labels
	query := s.buildGmailQueryUnified(options)

	// Set a higher limit per page for full sync with pagination support
	pageLimit := int64(500) // Increased from 100 to 500 per page
	totalLimit := int64(options.Limit)
	if totalLimit <= 0 {
		totalLimit = 1000 // Default total limit increased to 1000
	}

	s.logger.Debug("Fetching Gmail messages (unified) with query: %s, page_limit: %d, total_limit: %d", query, pageLimit, totalLimit)

	var allMessageRefs []*gmail.Message
	pageToken := ""
	pageCount := 0
	totalFetched := int64(0)

	// Paginate through all message lists
	for {
		listCall := service.Users.Messages.List("me").Q(query).MaxResults(pageLimit)
		if pageToken != "" {
			listCall = listCall.PageToken(pageToken)
		}

		listResp, err := listCall.Do()
		if err != nil {
			s.logger.Error("Failed to list Gmail messages on page %d: %v", pageCount+1, err)
			return nil, fmt.Errorf("failed to list Gmail messages: %w", err)
		}

		pageCount++
		s.logger.Debug("Gmail Messages List API Response (page %d):", pageCount)
		s.logger.Debug("  - Messages count: %d", len(listResp.Messages))
		s.logger.Debug("  - Next Page Token: %s", listResp.NextPageToken)

		if len(listResp.Messages) == 0 {
			break
		}

		// Add messages from this page
		for _, msgRef := range listResp.Messages {
			if totalFetched >= totalLimit {
				s.logger.Info("Reached total limit of %d messages", totalLimit)
				goto fetchDetails
			}
			allMessageRefs = append(allMessageRefs, &gmail.Message{Id: msgRef.Id})
			totalFetched++
		}

		// Check if there are more pages
		if listResp.NextPageToken == "" {
			break
		}
		pageToken = listResp.NextPageToken

		// Safety check to prevent infinite loops
		if pageCount >= 50 {
			s.logger.Warn("Reached maximum page limit (50) for Messages List API, stopping pagination")
			break
		}
	}

fetchDetails:
	s.logger.Debug("Processed %d pages from Gmail Messages List API, collected %d message IDs", pageCount, len(allMessageRefs))

	if len(allMessageRefs) == 0 {
		s.logger.Debug("No messages found in unified full sync")
		return []*gmail.Message{}, nil
	}

	// Fetch full message details for all collected messages
	var messages []*gmail.Message
	for i, msgRef := range allMessageRefs {
		msg, err := service.Users.Messages.Get("me", msgRef.Id).Do()
		if err != nil {
			s.logger.Warn("Failed to get message %s: %v", msgRef.Id, err)
			continue
		}
		messages = append(messages, msg)

		// Log progress for large batches
		if i > 0 && i%100 == 0 {
			s.logger.Info("Fetched details for %d/%d messages...", i, len(allMessageRefs))
		}
	}

	s.logger.Debug("Gmail unified full sync: fetched %d messages across %d pages", len(messages), pageCount)
	return messages, nil
}

// buildGmailQueryUnified builds Gmail search query without label filters
func (s *FetcherService) buildGmailQueryUnified(options FetchEmailsOptions) string {
	var queryParts []string

	// Date filter
	if options.StartDate != nil {
		queryParts = append(queryParts, fmt.Sprintf("after:%s", options.StartDate.Format("2006/01/02")))
	}
	if options.EndDate != nil {
		queryParts = append(queryParts, fmt.Sprintf("before:%s", options.EndDate.Format("2006/01/02")))
	}

	// Search query
	if options.SearchQuery != "" {
		queryParts = append(queryParts, options.SearchQuery)
	}

	query := strings.Join(queryParts, " ")
	if query == "" {
		query = "in:anywhere" // Get everything
	}

	return query
}

// filterGmailMessagesUnified applies unified filtering (date, search, NOT labels)
func (s *FetcherService) filterGmailMessagesUnified(messages []*gmail.Message, options FetchEmailsOptions) []*gmail.Message {
	var filtered []*gmail.Message

	for _, msg := range messages {
		// Apply date filter
		if options.StartDate != nil || options.EndDate != nil {
			msgDate := s.getGmailMessageDate(msg)
			if msgDate.IsZero() {
				continue // Skip messages without valid date
			}

			if options.StartDate != nil && msgDate.Before(*options.StartDate) {
				continue
			}
			if options.EndDate != nil && msgDate.After(*options.EndDate) {
				continue
			}
		}

		// Apply search filter
		if options.SearchQuery != "" {
			if !s.messageMatchesQuery(msg, options.SearchQuery) {
				continue
			}
		}

		// No label filtering - we want all messages with their labels intact
		filtered = append(filtered, msg)
	}

	return filtered
}

// Helper function to match Gmail labels with mailbox names
func (s *FetcherService) matchGmailLabelToMailbox(mailboxName string, labels []*gmail.Label) (string, error) {
	for _, label := range labels {
		if strings.Contains(mailboxName, "重要邮件") || strings.Contains(mailboxName, "Important") {
			if label.Id == "IMPORTANT" {
				return label.Id, nil
			}
		}
		if strings.Contains(mailboxName, "草稿") || strings.Contains(mailboxName, "Drafts") {
			if label.Id == "DRAFT" {
				return label.Id, nil
			}
		}
		if strings.Contains(mailboxName, "垃圾邮件") || strings.Contains(mailboxName, "Spam") {
			if label.Id == "SPAM" {
				return label.Id, nil
			}
		}
		if strings.Contains(mailboxName, "回收站") || strings.Contains(mailboxName, "Trash") {
			if label.Id == "TRASH" {
				return label.Id, nil
			}
		}
	}

	// If still no match, return empty string (no label filter)
	s.logger.Warn("Could not find Gmail label for mailbox: %s", mailboxName)
	return "", fmt.Errorf("label not found for mailbox: %s", mailboxName)
}

// getGmailMailboxes retrieves all Gmail labels as mailboxes using Gmail API
func (s *FetcherService) getGmailMailboxes(account models.EmailAccount) ([]models.Mailbox, error) {
	s.logger.Debug("Getting Gmail mailboxes using Gmail API for account %s", account.EmailAddress)

	// Create Gmail API service
	gmailService, err := s.createGmailService(account)
	if err != nil {
		s.logger.Error("Failed to create Gmail service: %v", err)
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Get all labels from Gmail
	labelList, err := gmailService.Users.Labels.List("me").Do()
	if err != nil {
		s.logger.Error("Failed to get Gmail labels: %v", err)
		return nil, fmt.Errorf("failed to get Gmail labels: %w", err)
	}

	var mailboxes []models.Mailbox
	for _, label := range labelList.Labels {
		mailbox := models.Mailbox{
			Name:      label.Name,
			AccountID: account.ID,
			Delimiter: "/", // Gmail uses forward slash as delimiter
		}

		// Convert label type to flags
		switch label.Type {
		case "system":
			mailbox.Flags = append(mailbox.Flags, "\\System")
		case "user":
			mailbox.Flags = append(mailbox.Flags, "\\User")
		}

		// Add visibility flags
		if label.LabelListVisibility == "labelShow" {
			mailbox.Flags = append(mailbox.Flags, "\\Visible")
		}
		if label.MessageListVisibility == "show" {
			mailbox.Flags = append(mailbox.Flags, "\\MessageShow")
		}

		mailboxes = append(mailboxes, mailbox)
	}

	s.logger.Info("Successfully retrieved %d Gmail labels as mailboxes for account %s", len(mailboxes), account.EmailAddress)
	return mailboxes, nil
}

// GetAccountByEmail gets an email account by email address
func (s *FetcherService) GetAccountByEmail(emailAddress string) (*models.EmailAccount, error) {
	s.logger.Debug("Getting account by email: %s", emailAddress)

	// Use the repository to find the account
	accounts, err := s.accountRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	for _, account := range accounts {
		if account.EmailAddress == emailAddress {
			s.logger.Debug("Found account for email: %s", emailAddress)
			return &account, nil
		}
	}

	return nil, fmt.Errorf("account not found for email: %s", emailAddress)
}
