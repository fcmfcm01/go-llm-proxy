package testutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// AssertNoError asserts that err is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	assert.NoError(t, err)
}

// AssertError asserts that err is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	assert.Error(t, err)
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	assert.Equal(t, expected, actual)
}

// MockConfig returns a mock configuration for testing
func MockConfig() map[string]interface{} {
	return map[string]interface{}{
		"server": map[string]interface{}{
			"port": 8443,
			"host": "localhost",
		},
		"providers": []map[string]interface{}{},
	}
}
