package random

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		expected string
	}{
		{
			name:     "size = 1",
			size:     1,
			expected: alphabet,
		},
		{
			name:     "size = 5",
			size:     5,
			expected: alphabet,
		},
		{
			name:     "size = 10",
			size:     10,
			expected: alphabet,
		},
		{
			name:     "size = 20",
			size:     20,
			expected: alphabet,
		},
		{
			name:     "size = 30",
			size:     30,
			expected: alphabet,
		},
		{
			name:     "size = 0",
			size:     0,
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str1 := NewRandomString(tt.size)
			str2 := NewRandomString(tt.size)

			assert.Len(t, str1, tt.size)
			assert.Len(t, str2, tt.size)

			if tt.size > 0 {
				assert.NotEqual(t, str1, str2)
			} else {
				assert.Equal(t, str1, tt.expected)
				assert.Equal(t, str2, tt.expected)
			}

			for _, r := range str1 {
				assert.Contains(t, alphabet, string(r))
			}
		})
	}
}

func TestNewRandomString_CorrectSymbols(t *testing.T) {
	const size = 1000
	s := NewRandomString(size)

	require.Len(t, s, size)

	for _, char := range s {
		require.True(t, strings.Contains(alphabet, string(char)), "string contains invalid characters")
	}
}
