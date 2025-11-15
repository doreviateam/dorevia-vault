package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple object - tri des clés",
			input:    `{"b":2,"a":1}`,
			expected: `{"a":1,"b":2}`,
		},
		{
			name:     "with null - suppression null",
			input:    `{"a":1,"b":null,"c":3}`,
			expected: `{"a":1,"c":3}`,
		},
		{
			name:     "nested object - tri récursif",
			input:    `{"z":{"b":2,"a":1},"y":10}`,
			expected: `{"y":10,"z":{"a":1,"b":2}}`,
		},
		{
			name:     "number normalization - 10.0 → 10",
			input:    `{"a":10.0,"b":10.5}`,
			expected: `{"a":10,"b":10.5}`,
		},
		{
			name:     "array - normalisation éléments",
			input:    `{"items":[{"b":2,"a":1},{"d":4,"c":3}]}`,
			expected: `{"items":[{"a":1,"b":2},{"c":3,"d":4}]}`,
		},
		{
			name:     "complex nested - tous les cas",
			input:    `{"z":{"b":2.0,"a":1,"c":null},"y":10.0,"items":[{"x":5.0}]}`,
			expected: `{"items":[{"x":5}],"y":10,"z":{"a":1,"b":2}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalizeJSON([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestCanonicalizeJSON_Stability(t *testing.T) {
	// Test que le même contenu produit toujours le même hash
	inputs := []string{
		`{"b":2,"a":1,"c":null}`,
		`{"a":1,"b":2.0}`,
		`{"a":1.0,"b":2,"c":null}`,
	}

	var hashes []string
	for _, input := range inputs {
		canonical, err := CanonicalizeJSON([]byte(input))
		require.NoError(t, err)

		hash := sha256.Sum256(canonical)
		hashHex := hex.EncodeToString(hash[:])
		hashes = append(hashes, hashHex)
	}

	// Tous les inputs doivent produire le même hash après canonicalisation
	for i := 1; i < len(hashes); i++ {
		assert.Equal(t, hashes[0], hashes[i], "Hash should be stable across different JSON representations")
	}
}

func TestCanonicalizeJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty object",
			input: `{}`,
		},
		{
			name:  "empty array",
			input: `{"items":[]}`,
		},
		{
			name:  "all null values",
			input: `{"a":null,"b":null}`,
		},
		{
			name:  "nested null",
			input: `{"a":{"b":null},"c":1}`,
		},
		{
			name:  "array with null",
			input: `{"items":[1,null,2]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CanonicalizeJSON([]byte(tt.input))
			require.NoError(t, err)
			// Vérifier que le résultat est valide JSON
			var check interface{}
			err = json.Unmarshal(result, &check)
			assert.NoError(t, err, "Result should be valid JSON")
		})
	}
}

func TestCanonicalizeJSON_InvalidJSON(t *testing.T) {
	invalidInputs := []string{
		`{invalid json}`,
		`{"a":}`,
		`{"a":1,}`,
	}

	for _, input := range invalidInputs {
		_, err := CanonicalizeJSON([]byte(input))
		assert.Error(t, err, "Should return error for invalid JSON")
	}
}

