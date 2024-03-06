package utils

import "testing"

func TestDisplayContainerName(t *testing.T) {
	testCases := []struct {
		name      string
		stack     string
		container string
		expected  string
	}{
		{
			name:      "No stack",
			stack:     "",
			container: "container-123",
			expected:  "container-123",
		},
		{
			name:      "Stack",
			stack:     "stack",
			container: "stack-container-123",
			expected:  "container-123",
		},
		{
			name:      "Stack with number",
			stack:     "stack",
			container: "stack-container-12-3",
			expected:  "container-12",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := DisplayContainerName(testCase.container, testCase.stack); got != testCase.expected {
				t.Errorf("unexpected container name, got %v, want %v", got, testCase.expected)
			}
		})
	}
}
