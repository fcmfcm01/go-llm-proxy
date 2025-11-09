package middleware

import (
	"bytes"
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// InputSanitizer provides input sanitization methods
type InputSanitizer struct {
	stripTagsRegex *regexp.Regexp
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer() *InputSanitizer {
	// Regex to strip HTML tags
	stripTags, _ := regexp.Compile("<[^>]*>")

	return &InputSanitizer{
		stripTagsRegex: stripTags,
	}
}

// SanitizeString removes or escapes potentially dangerous characters
func (s *InputSanitizer) SanitizeString(input string) string {
	// Escape HTML
	escaped := html.EscapeString(input)

	// Remove null bytes
	cleaned := strings.ReplaceAll(escaped, "\x00", "")

	return cleaned
}

// SanitizeHTML allows only specific HTML tags and attributes
func (s *InputSanitizer) SanitizeHTML(input string) string {
	// This is a simplified implementation
	// In production, use a library like bluemonday

	allowedTags := map[string]bool{
		"p":      true,
		"br":     true,
		"strong": true,
		"em":     true,
		"b":      true,
		"i":      true,
	}

	// Remove disallowed tags
	result := s.stripTagsRegex.ReplaceAllStringFunc(input, func(tag string) string {
		tagName := tag[1 : len(tag)-1] // Remove < and >
		if slash := strings.Index(tagName, "/"); slash != -1 {
			tagName = tagName[slash+1:]
		}
		if allowedTags[tagName] {
			return tag
		}
		return ""
	})

	return html.EscapeString(result)
}

// StripHTML removes all HTML tags
func (s *InputSanitizer) StripHTML(input string) string {
	return s.stripTagsRegex.ReplaceAllString(input, "")
}

// ValidateInput validates input based on rules
func (s *InputSanitizer) ValidateInput(input string, rules ValidationRules) error {
	// Check length
	if rules.MinLength > 0 && len(input) < rules.MinLength {
		return gin.Error{Err: nil, Type: gin.ErrorTypeBind, Meta: "input too short"}
	}
	if rules.MaxLength > 0 && len(input) > rules.MaxLength {
		return gin.Error{Err: nil, Type: gin.ErrorTypeBind, Meta: "input too long"}
	}

	// Check pattern
	if rules.Pattern != "" {
		matched, _ := regexp.MatchString(rules.Pattern, input)
		if !matched {
			return gin.Error{Err: nil, Type: gin.ErrorTypeBind, Meta: "input format invalid"}
		}
	}

	// Check for SQL injection patterns
	sqlPatterns := []string{
		"(?i)union(?-i) select",
		"(?i)drop table",
		"(?i)delete from",
		"(?i)insert into",
		"(?i)update set",
		"' or '1'='1",
		"\" or \"1\"=\"1",
	}

	for _, pattern := range sqlPatterns {
		matched, _ := regexp.MatchString(pattern, input)
		if matched {
			return gin.Error{Err: nil, Type: gin.ErrorTypeBind, Meta: "potentially dangerous input"}
		}
	}

	return nil
}

// ValidationRules holds validation rules for input
type ValidationRules struct {
	MinLength int
	MaxLength int
	Pattern   string
	AllowHTML bool
	Required  bool
}

// SanitizeRequestMiddleware sanitizes request parameters and body
func SanitizeRequestMiddleware() gin.HandlerFunc {
	sanitizer := NewInputSanitizer()

	return func(c *gin.Context) {
		// Sanitize query parameters
		for key, values := range c.Request.URL.Query() {
			for i, value := range values {
				c.Request.URL.Query()[key][i] = sanitizer.SanitizeString(value)
			}
		}

		// Sanitize form data
		c.Request.ParseForm()
		for key, values := range c.Request.Form {
			for i, value := range values {
				c.Request.Form[key][i] = sanitizer.SanitizeString(value)
			}
		}

		// Add sanitized values to context for handlers to use
		c.Set("sanitizer", sanitizer)

		c.Next()
	}
}

// ValidateJSONSchema validates JSON input against a schema (simplified)
func ValidateJSONSchema(schema map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a simplified validation
		// In production, use a JSON schema validator library

		// Check Content-Type
		contentType := c.GetHeader("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			c.JSON(415, gin.H{
				"error": "Unsupported Media Type",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PreventXSSMiddleware prevents XSS attacks
func PreventXSSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set XSS protection header
		c.Header("X-XSS-Protection", "1; mode=block")

		// Validate content type
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "application/json") {
				// Sanitize JSON body if needed
				var body bytes.Buffer
				body.ReadFrom(c.Request.Body)

				// Basic XSS check
				xssPatterns := []string{
					"<script",
					"javascript:",
					"vbscript:",
					"onload=",
					"onerror=",
				}

				bodyStr := body.String()
				for _, pattern := range xssPatterns {
					if strings.Contains(strings.ToLower(bodyStr), pattern) {
						c.JSON(400, gin.H{
							"error": "Potentially dangerous content detected",
						})
						c.Abort()
						return
					}
				}
			}
		}

		c.Next()
	}
}

// EscapeTemplate escapes template variables automatically
func EscapeTemplate(c *gin.Context, key string, value interface{}) {
	if str, ok := value.(string); ok {
		c.Set(key, template.HTMLEscapeString(str))
	} else {
		c.Set(key, value)
	}
}
