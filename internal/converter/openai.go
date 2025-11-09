package converter

// OpenAIConverter handles OpenAI format conversions
type OpenAIConverter struct{}

// NewOpenAIConverter creates a new OpenAI converter
func NewOpenAIConverter() *OpenAIConverter {
	return &OpenAIConverter{}
}

// ConvertRequest validates and normalizes OpenAI requests
func (c *OpenAIConverter) ConvertRequest(req map[string]interface{}) (map[string]interface{}, error) {
	// OpenAI format is the standard, so just validate and return
	normalized := make(map[string]interface{})

	// Copy all fields
	for k, v := range req {
		normalized[k] = v
	}

	return normalized, nil
}

// ConvertResponse validates and normalizes OpenAI responses
func (c *OpenAIConverter) ConvertResponse(resp map[string]interface{}) (map[string]interface{}, error) {
	// OpenAI format is the standard
	return resp, nil
}
