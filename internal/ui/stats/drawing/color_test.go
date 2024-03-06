package drawing

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestIsHexColor(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid hex color",
			input:    "#FF0000",
			expected: true,
		},
		{
			name:     "Valid hex color with alpha",
			input:    "#7F0000FF",
			expected: true,
		},
		{
			name:     "Invalid length",
			input:    "FF0000",
			expected: false,
		},
		{
			name:     "Invalid length with alpha",
			input:    "7F0000",
			expected: false,
		},
		{
			name:     "Invalid character",
			input:    "#GF0000",
			expected: false,
		},
		{
			name:     "Invalid character with alpha",
			input:    "#7FG000FF",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := isHexColor(testCase.input)
			if actual != testCase.expected {
				t.Errorf("expected %v, got %v", testCase.expected, actual)
			}
		})
	}
}

func TestLipglossToRGBA(t *testing.T) {
	testCases := []struct {
		name     string
		input    lipgloss.Color
		expected color.RGBA
	}{
		{
			name:     "Valid 6-digit hex color",
			input:    lipgloss.Color("#FFAABB"),
			expected: color.RGBA{R: 255, G: 170, B: 187, A: 225},
		},
		{
			name:     "Valid 8-digit hex color",
			input:    lipgloss.Color("#FFAABBCC"),
			expected: color.RGBA{R: 255, G: 170, B: 187, A: 204},
		},
		{
			name:  "Invalid hex color",
			input: lipgloss.Color("invalid"),
			// error case, no expected RGBA
		},
		{
			name:  "Invalid hex color format",
			input: lipgloss.Color("#12345"),
			// error case, no expected RGBA
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := lipglossToRGBA(tc.input)
			if err != nil {
				if tc.expected != (color.RGBA{}) {
					t.Errorf("unexpected error for test case %q: %v", tc.name, err)
				}
			} else if result != tc.expected {
				t.Errorf("expected: %v, Got: %v", tc.expected, result)
			}
		})
	}
}

func TestGenerateColorGradient(t *testing.T) {
	testCases := []struct {
		Start    lipgloss.Color
		End      lipgloss.Color
		Steps    int
		Expected []lipgloss.Color
	}{
		{
			Start:    lipgloss.Color("#FFFFFF"),
			End:      lipgloss.Color("#FF0000"),
			Steps:    5,
			Expected: []lipgloss.Color{lipgloss.Color("#FFFFFF"), lipgloss.Color("#FFBFBF"), lipgloss.Color("#FF7F7F"), lipgloss.Color("#FF3F3F"), lipgloss.Color("#FF0000")},
		},
		{
			Start:    lipgloss.Color("#FFFFFF"),
			End:      lipgloss.Color("#000000"),
			Steps:    5,
			Expected: []lipgloss.Color{lipgloss.Color("#FFFFFF"), lipgloss.Color("#BFBFBF"), lipgloss.Color("#7F7F7F"), lipgloss.Color("#3F3F3F"), lipgloss.Color("#000000")},
		},
		{
			Start:    lipgloss.Color("#D7DCFF"),
			End:      lipgloss.Color("#123456"),
			Steps:    4,
			Expected: []lipgloss.Color{lipgloss.Color("#D7DCFF"), lipgloss.Color("#95A4C6"), lipgloss.Color("#536C8E"), lipgloss.Color("#123456")},
		},
		{
			Start:    lipgloss.Color("#C30211"),
			End:      lipgloss.Color("#54E316"),
			Steps:    7,
			Expected: []lipgloss.Color{lipgloss.Color("#C30211"), lipgloss.Color("#B02711"), lipgloss.Color("#9E4D12"), lipgloss.Color("#8B7213"), lipgloss.Color("#799814"), lipgloss.Color("#66BD15"), lipgloss.Color("#54E316")},
		},
	}

	for _, testCase := range testCases {
		actual := generateColorGradient(testCase.Start, testCase.End, testCase.Steps)
		if len(actual) != len(testCase.Expected) || !equalGradient(actual, testCase.Expected) {
			t.Errorf("expected %v, got %v", testCase.Expected, actual)
		}
	}
}

func equalGradient(a, b []lipgloss.Color) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalColors(a[i], b[i]) {
			return false
		}
	}
	return true
}

func equalColors(firstColor, secondColor lipgloss.Color) bool {
	a, e := lipglossToRGBA(firstColor)
	if e != nil {
		return false
	}

	b, e := lipglossToRGBA(secondColor)
	if e != nil {
		return false
	}

	return a == b
}
