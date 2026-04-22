package email

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseEmailContent extracts clean text from email HTML or plain text
// Priority: HTML content > Plain text > Empty string
func ParseEmailContent(htmlBody, plainBody string) (string, error) {
	// Prefer HTML as it usually has better formatting
	if htmlBody != "" {
		return ParseHTML(htmlBody)
	}

	// Fallback to plain text
	if plainBody != "" {
		return CleanPlainText(plainBody), nil
	}

	return "", nil
}

// ParseHTML converts HTML email content to clean text
func ParseHTML(html string) (string, error) {
	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove unwanted elements
	doc.Find("script, style, img, svg, noscript").Remove()

	// Remove common newsletter footer elements
	doc.Find("[class*='footer'], [class*='unsubscribe'], [id*='footer'], [id*='unsubscribe']").Remove()

	// Remove tracking pixels and invisible elements
	doc.Find("[style*='display:none'], [style*='display: none'], [style*='visibility:hidden']").Remove()

	// Remove common button/CTA sections (often just "Read More" links)
	doc.Find("button, [class*='cta'], [class*='button']").Remove()

	// Extract text
	text := doc.Text()

	// Clean up the text
	return CleanText(text), nil
}

// CleanPlainText cleans up plain text email content
func CleanPlainText(text string) string {
	// Remove common unsubscribe footers
	text = removeUnsubscribeFooter(text)

	// Remove excessive whitespace
	text = CleanText(text)

	return text
}

// CleanText performs general text cleanup
func CleanText(text string) string {
	// Remove excessive whitespace
	text = strings.TrimSpace(text)

	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace more than 2 consecutive newlines with 2 newlines
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	// Remove leading/trailing whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	return text
}

// removeUnsubscribeFooter removes common unsubscribe footer patterns
func removeUnsubscribeFooter(text string) string {
	// Common unsubscribe patterns
	patterns := []string{
		`(?i)unsubscribe.*`,
		`(?i)if you.*?no longer.*?receive.*`,
		`(?i)you.*?receiving this.*?because.*`,
		`(?i)click here to.*?(unsubscribe|opt[- ]out).*`,
		`(?i)to stop receiving.*`,
		`(?i)manage.*?preferences.*`,
		`(?i)©.*?\d{4}.*`, // Copyright notices
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		text = re.ReplaceAllString(text, "")
	}

	return text
}

// MatchesSenderPattern checks if an email address matches a pattern
// Patterns can be:
//   - Exact match: "newsletter@example.com"
//   - Domain wildcard: "*@substack.com"
//   - Substring: "*newsletter*" (matches any email with "newsletter")
func MatchesSenderPattern(email, pattern string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Exact match
	if email == pattern {
		return true
	}

	// Domain wildcard: *@substack.com
	if after, ok := strings.CutPrefix(pattern, "*@"); ok {
		domain := after
		return strings.HasSuffix(email, "@"+domain)
	}

	// Substring match: *newsletter*
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substring := strings.Trim(pattern, "*")
		return strings.Contains(email, substring)
	}

	// Prefix match: newsletter*
	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		prefix := before
		return strings.HasPrefix(email, prefix)
	}

	// Suffix match: *newsletter
	if after, ok := strings.CutPrefix(pattern, "*"); ok {
		suffix := after
		return strings.HasSuffix(email, suffix)
	}

	return false
}

// MatchesSubjectPattern checks if a subject matches a regex pattern
// Returns true if pattern is empty/nil or if subject matches the pattern
func MatchesSubjectPattern(subject string, pattern *string) bool {
	if pattern == nil || *pattern == "" {
		return true
	}

	re, err := regexp.Compile(*pattern)
	if err != nil {
		// If pattern is invalid, treat as no match
		return false
	}

	return re.MatchString(subject)
}

// ExtractSenderEmail extracts the email address from a "From" header
// Example: "John Doe <john@example.com>" -> "john@example.com"
func ExtractSenderEmail(from string) string {
	// Look for email in angle brackets
	re := regexp.MustCompile(`<([^>]+)>`)
	matches := re.FindStringSubmatch(from)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no angle brackets, assume the whole string is an email
	return strings.TrimSpace(from)
}
