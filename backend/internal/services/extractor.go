package services

import (
	"encoding/json"
	"fmt"
	"mailman/internal/models"
	"regexp"
	"strings"
	"text/template"

	"github.com/robertkrimen/otto"
)

// ExtractorType defines the type of extraction to perform
type ExtractorType string

const (
	ExtractorTypeRegex      ExtractorType = "regex"
	ExtractorTypeJS         ExtractorType = "js"
	ExtractorTypeGoTemplate ExtractorType = "gotemplate"
)

// ExtractorField defines which field to extract from
type ExtractorField string

const (
	ExtractorFieldAll      ExtractorField = "ALL"
	ExtractorFieldFrom     ExtractorField = "from"
	ExtractorFieldTo       ExtractorField = "to"
	ExtractorFieldCC       ExtractorField = "cc"
	ExtractorFieldSubject  ExtractorField = "subject"
	ExtractorFieldBody     ExtractorField = "body"
	ExtractorFieldHTMLBody ExtractorField = "html_body"
	ExtractorFieldHeaders  ExtractorField = "headers"
)

// ExtractorConfig defines the configuration for content extraction
type ExtractorConfig struct {
	Field   ExtractorField `json:"field"`
	Type    ExtractorType  `json:"type"`
	Match   *string        `json:"match,omitempty"` // Optional match configuration
	Extract string         `json:"extract"`         // Extract configuration
}

// MatchResult represents the result of a match operation
type MatchResult struct {
	Matched bool   `json:"matched"`
	Reason  string `json:"reason,omitempty"`
}

// ExtractorResult represents the result of an extraction operation
type ExtractorResult struct {
	Email   models.Email `json:"email"`
	Matches []string     `json:"matches"`
}

// ExtractorService handles email content extraction
type ExtractorService struct{}

// NewExtractorService creates a new ExtractorService
func NewExtractorService() *ExtractorService {
	return &ExtractorService{}
}

// ExtractFromEmail extracts content from a single email using the provided extractors
func (s *ExtractorService) ExtractFromEmail(email models.Email, extractors []ExtractorConfig) (*ExtractorResult, error) {
	var allMatches []string
	hasMatch := false

	for _, extractor := range extractors {
		// First check if we need to evaluate match condition
		if extractor.Match != nil {
			matchResult, err := s.evaluateMatch(email, extractor)
			if err != nil {
				return nil, fmt.Errorf("match evaluation failed for field %s: %w", extractor.Field, err)
			}
			if !matchResult.Matched {
				// Skip extraction if match condition is not met
				continue
			}
		}

		// Perform extraction
		matches, err := s.extractWithConfig(email, extractor)
		if err != nil {
			return nil, fmt.Errorf("extraction failed for field %s with type %s: %w", extractor.Field, extractor.Type, err)
		}

		if len(matches) > 0 {
			hasMatch = true
			allMatches = append(allMatches, matches...)
		}
	}

	// Only return result if there were matches
	if !hasMatch {
		return nil, nil
	}

	return &ExtractorResult{
		Email:   email,
		Matches: allMatches,
	}, nil
}

// evaluateMatch evaluates the match condition
func (s *ExtractorService) evaluateMatch(email models.Email, config ExtractorConfig) (*MatchResult, error) {
	if config.Match == nil {
		// Default to matched if no match condition is provided
		return &MatchResult{Matched: true}, nil
	}

	// Get the field content to match against
	fieldContent := s.getFieldContent(email, config.Field)

	switch config.Type {
	case ExtractorTypeRegex:
		return s.matchWithRegex(fieldContent, *config.Match)
	case ExtractorTypeJS:
		return s.matchWithJS(email, fieldContent, *config.Match)
	case ExtractorTypeGoTemplate:
		return s.matchWithGoTemplate(email, *config.Match)
	default:
		return nil, fmt.Errorf("unsupported extractor type for match: %s", config.Type)
	}
}

// matchWithRegex performs regex-based matching
func (s *ExtractorService) matchWithRegex(content []string, pattern string) (*MatchResult, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	for _, text := range content {
		if text == "" {
			continue
		}
		if regex.MatchString(text) {
			return &MatchResult{Matched: true}, nil
		}
	}

	return &MatchResult{
		Matched: false,
		Reason:  "Pattern not found in content",
	}, nil
}

// matchWithJS performs JavaScript-based matching
func (s *ExtractorService) matchWithJS(email models.Email, content []string, script string) (*MatchResult, error) {
	vm := otto.New()

	// Set up the content variable in the JS context
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	// Set the content variable
	err = vm.Set("content", string(contentJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to set content variable: %w", err)
	}

	// Set the email object
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email: %w", err)
	}
	err = vm.Set("email", string(emailJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to set email variable: %w", err)
	}

	// Parse content and email as JSON in JS
	_, err = vm.Run(`
		var parsedContent = JSON.parse(content);
		var parsedEmail = JSON.parse(email);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data in JS: %w", err)
	}

	// Wrap the script to ensure it returns a proper match result
	wrappedScript := fmt.Sprintf(`
		(function() {
			try {
				var result = (function() { %s })();
				if (typeof result === 'boolean') {
					return JSON.stringify({matched: result});
				} else if (typeof result === 'object' && result !== null && 'matched' in result) {
					return JSON.stringify(result);
				} else {
					return JSON.stringify({matched: false, reason: 'Invalid return value from match script'});
				}
			} catch (e) {
				return JSON.stringify({matched: false, reason: e.toString()});
			}
		})()
	`, script)

	// Execute the wrapped script
	result, err := vm.Run(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("JS execution failed: %w", err)
	}

	// Convert result to string
	resultStr, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert JS result to string: %w", err)
	}

	// Parse the result
	var matchResult MatchResult
	if err := json.Unmarshal([]byte(resultStr), &matchResult); err != nil {
		return nil, fmt.Errorf("failed to parse match result: %w", err)
	}

	return &matchResult, nil
}

// matchWithGoTemplate performs Go template-based matching
func (s *ExtractorService) matchWithGoTemplate(email models.Email, templateStr string) (*MatchResult, error) {
	// Define custom template functions
	funcMap := template.FuncMap{
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"toLower":   strings.ToLower,
		"toUpper":   strings.ToUpper,
		"trim":      strings.TrimSpace,
		"split":     strings.Split,
		"join":      strings.Join,
		"replace":   strings.ReplaceAll,
	}

	tmpl, err := template.New("matcher").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Create a data structure that includes the email
	data := struct {
		*models.Email
		AllText string
	}{
		Email:   &email,
		AllText: strings.Join([]string{email.Subject, email.Body, email.HTMLBody}, " "),
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	output := strings.TrimSpace(result.String())

	// Try to parse as JSON first
	var matchResult MatchResult
	if err := json.Unmarshal([]byte(output), &matchResult); err == nil {
		return &matchResult, nil
	}

	// If not JSON, treat as boolean string
	matched := output == "true" || output == "1" || output == "yes"
	reason := ""
	if !matched && output != "false" && output != "0" && output != "no" && output != "" {
		reason = output
	}

	return &MatchResult{
		Matched: matched,
		Reason:  reason,
	}, nil
}

// extractWithConfig performs extraction based on the configuration
func (s *ExtractorService) extractWithConfig(email models.Email, config ExtractorConfig) ([]string, error) {
	// Get the field content to extract from
	fieldContent := s.getFieldContent(email, config.Field)

	switch config.Type {
	case ExtractorTypeRegex:
		return s.extractWithRegex(fieldContent, config.Extract)
	case ExtractorTypeJS:
		return s.extractWithJS(email, fieldContent, config.Extract)
	case ExtractorTypeGoTemplate:
		return s.extractWithGoTemplate(email, config.Extract)
	default:
		return nil, fmt.Errorf("unsupported extractor type: %s", config.Type)
	}
}

// getFieldContent extracts the content from the specified field
func (s *ExtractorService) getFieldContent(email models.Email, field ExtractorField) []string {
	switch field {
	case ExtractorFieldFrom:
		return email.From
	case ExtractorFieldTo:
		return email.To
	case ExtractorFieldCC:
		return email.Cc
	case ExtractorFieldSubject:
		return []string{email.Subject}
	case ExtractorFieldBody:
		return []string{email.Body}
	case ExtractorFieldHTMLBody:
		return []string{email.HTMLBody}
	case ExtractorFieldHeaders:
		// For headers, we would need to add a Headers field to the Email model
		// For now, return empty
		return []string{}
	case ExtractorFieldAll:
		// Combine all text fields
		var all []string
		all = append(all, email.From...)
		all = append(all, email.To...)
		all = append(all, email.Cc...)
		all = append(all, email.Subject)
		all = append(all, email.Body)
		all = append(all, email.HTMLBody)
		return all
	default:
		return []string{}
	}
}

// extractWithRegex performs regex-based extraction with support for $0...$n replacements
func (s *ExtractorService) extractWithRegex(content []string, pattern string) ([]string, error) {
	// Check if pattern contains replacement syntax ($0, $1, etc.)
	parts := strings.SplitN(pattern, "|||", 2)
	regexPattern := parts[0]
	replacement := ""
	if len(parts) > 1 {
		replacement = parts[1]
	}

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []string
	for _, text := range content {
		if text == "" {
			continue
		}

		if replacement != "" {
			// Use replacement syntax
			allMatches := regex.FindAllStringSubmatch(text, -1)
			for _, match := range allMatches {
				result := replacement
				// Replace $0, $1, $2, etc. with captured groups
				for i, group := range match {
					placeholder := fmt.Sprintf("$%d", i)
					result = strings.ReplaceAll(result, placeholder, group)
				}
				if result != "" {
					matches = append(matches, result)
				}
			}
		} else {
			// Standard extraction without replacement
			found := regex.FindAllString(text, -1)
			matches = append(matches, found...)
		}
	}

	return matches, nil
}

// extractWithJS performs JavaScript-based extraction
func (s *ExtractorService) extractWithJS(email models.Email, content []string, script string) ([]string, error) {
	vm := otto.New()

	// Set up the content variable in the JS context
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content: %w", err)
	}

	// Set the content variable
	err = vm.Set("content", string(contentJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to set content variable: %w", err)
	}

	// Set the email object
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal email: %w", err)
	}
	err = vm.Set("email", string(emailJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to set email variable: %w", err)
	}

	// Parse content and email as JSON in JS
	_, err = vm.Run(`
		var parsedContent = JSON.parse(content);
		var parsedEmail = JSON.parse(email);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data in JS: %w", err)
	}

	// Wrap the script to ensure it returns a proper result
	wrappedScript := fmt.Sprintf(`
		(function() {
			try {
				var result = (function() { %s })();
				if (result === null || result === undefined) {
					return JSON.stringify([]);
				} else if (typeof result === 'string') {
					return JSON.stringify([result]);
				} else if (Array.isArray(result)) {
					return JSON.stringify(result.filter(function(item) { 
						return item !== null && item !== undefined && item !== ''; 
					}));
				} else {
					return JSON.stringify([String(result)]);
				}
			} catch (e) {
				return JSON.stringify([]);
			}
		})()
	`, script)

	// Execute the wrapped script
	result, err := vm.Run(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("JS execution failed: %w", err)
	}

	// Convert result to string
	resultStr, err := result.ToString()
	if err != nil {
		return nil, fmt.Errorf("failed to convert JS result to string: %w", err)
	}

	// Parse as JSON array
	var matches []string
	if err := json.Unmarshal([]byte(resultStr), &matches); err != nil {
		return nil, fmt.Errorf("failed to parse extraction result: %w", err)
	}

	return matches, nil
}

// extractWithGoTemplate performs Go template-based extraction
func (s *ExtractorService) extractWithGoTemplate(email models.Email, templateStr string) ([]string, error) {
	// Define custom template functions
	funcMap := template.FuncMap{
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"toLower":   strings.ToLower,
		"toUpper":   strings.ToUpper,
		"trim":      strings.TrimSpace,
		"split":     strings.Split,
		"join":      strings.Join,
		"replace":   strings.ReplaceAll,
		"extractLinks": func(text string) []string {
			// Simple regex to extract URLs
			urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]` + "`" + `]+`)
			return urlRegex.FindAllString(text, -1)
		},
		"extractEmails": func(text string) []string {
			// Simple regex to extract email addresses
			emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
			return emailRegex.FindAllString(text, -1)
		},
		"regex": func(pattern, text string) []string {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return []string{}
			}
			return regex.FindAllString(text, -1)
		},
		"regexReplace": func(pattern, replacement, text string) string {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return text
			}
			return regex.ReplaceAllString(text, replacement)
		},
	}

	tmpl, err := template.New("extractor").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Create a data structure that includes the email and helper fields
	data := struct {
		*models.Email
		AllText string
		Links   []string
		Emails  []string
	}{
		Email:   &email,
		AllText: strings.Join([]string{email.Subject, strings.Join(email.From, " "), strings.Join(email.To, " "), strings.Join(email.Cc, " "), email.Body, email.HTMLBody}, " "),
		Links:   extractLinksFromEmail(email),
		Emails:  extractEmailsFromEmail(email),
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}

	output := strings.TrimSpace(result.String())
	if output == "" {
		return []string{}, nil
	}

	// Try to parse as JSON array first
	var matches []string
	if err := json.Unmarshal([]byte(output), &matches); err == nil {
		// Filter out empty strings
		var filtered []string
		for _, match := range matches {
			if match != "" {
				filtered = append(filtered, match)
			}
		}
		return filtered, nil
	}

	// If not JSON array, return as single string
	return []string{output}, nil
}

// extractLinksFromEmail extracts all links from email content
func extractLinksFromEmail(email models.Email) []string {
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]` + "`" + `]+`)
	var links []string

	// Extract from all text fields
	allText := strings.Join([]string{email.Subject, email.Body, email.HTMLBody}, " ")
	links = urlRegex.FindAllString(allText, -1)

	// Remove duplicates
	linkMap := make(map[string]bool)
	var uniqueLinks []string
	for _, link := range links {
		if !linkMap[link] {
			linkMap[link] = true
			uniqueLinks = append(uniqueLinks, link)
		}
	}

	return uniqueLinks
}

// extractEmailsFromEmail extracts all email addresses from email content
func extractEmailsFromEmail(email models.Email) []string {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	var emails []string

	// Extract from all text fields
	allText := strings.Join([]string{email.Subject, email.Body, email.HTMLBody}, " ")
	emails = emailRegex.FindAllString(allText, -1)

	// Also include From, To, CC
	emails = append(emails, email.From...)
	emails = append(emails, email.To...)
	emails = append(emails, email.Cc...)

	// Remove duplicates
	emailMap := make(map[string]bool)
	var uniqueEmails []string
	for _, e := range emails {
		if !emailMap[e] {
			emailMap[e] = true
			uniqueEmails = append(uniqueEmails, e)
		}
	}

	return uniqueEmails
}
