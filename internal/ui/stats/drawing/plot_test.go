package drawing

import (
	"math"
	"testing"
)

func TestConvertToBrailleRuneIndex(t *testing.T) {
	testCases := []struct {
		value, scale          float64
		expectedIndex         int
		expectedAdjustedValue float64
	}{
		{4.0, 1.0, 4, 0.0},
		{3.99, 1.0, 3, 0.0},
		{3.0, 1.0, 3, 0.0},
		{2.99, 1.0, 2, 0.0},
		{2.0, 1.0, 2, 0.0},
		{1.99, 1.0, 1, 0.0},
		{1.0, 1.0, 1, 0.0},
		{0.99, 1.0, 1, 0.0},
		{0.0, 1.0, 0, 0.0},
		{0.0001, 1.0, 1, 0.0},
		{0.0, 0.5, 0, 0.0},
		{0.5, 0.5, 1, 0.0},
		{0.5001, 0.5, 1, 0.0},
		{1.0, 0.5, 2, 0.0},
		{1.5, 0.5, 3, 0},
		{1.99, 0.5, 3, 0},
		{2.0, 0.5, 4, 0.0},
		{2.5, 0.5, 4, 0.5},
		{2.99, 0.5, 4, 0.99},
		{3.0, 0.5, 4, 1.0},
		{3.5, 0.5, 4, 1.5},
	}

	for _, testCase := range testCases {
		index, adjustedValue := convertToBrailleRuneIndex(testCase.value, testCase.scale)

		if index != testCase.expectedIndex {
			t.Errorf("unexpected index, got: %v, expected: %v", index, testCase.expectedIndex)
		}

		if math.Abs(adjustedValue-testCase.expectedAdjustedValue) > 0.001 {
			t.Errorf("unexpected adjusted value, got: %v, expected: %v", adjustedValue, testCase.expectedAdjustedValue)
		}
	}
}

func TestPlotPush(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		width       int
		pushValues  []float64
		expected    []float64
		expectedMax float64
	}{
		{
			name:        "basic",
			width:       5,
			pushValues:  []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			expected:    []float64{15, 14, 13, 12, 11, 10, 9, 8, 7, 6},
			expectedMax: 15,
		},
		{
			name:        "empty",
			width:       5,
			pushValues:  []float64{},
			expected:    []float64{},
			expectedMax: 0,
		},
		{
			name:        "single value",
			width:       5,
			pushValues:  []float64{1},
			expected:    []float64{1},
			expectedMax: 1,
		},
	}

	for _, tc := range testCases {
		testCases := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plot := New[float64](ColorGradient{})
			plot.SetSize(testCases.width, 100)

			for _, v := range testCases.pushValues {
				plot.Push(v)
			}

			actual := plot.data.ToArray()

			if len(actual) != len(testCases.expected) {
				t.Errorf("unexpected length, got: %v, expected: %v", len(actual), len(testCases.expected))
				return
			}

			for i, v := range actual {
				if v != testCases.expected[i] {
					t.Errorf("unexpected value, got: %v, expected: %v", v, testCases.expected[i])
				}
			}

			if plot.maxValue != testCases.expectedMax {
				t.Errorf("unexpected max value, got: %v, expected: %v", plot.maxValue, testCases.expectedMax)
			}
		})
	}
}
