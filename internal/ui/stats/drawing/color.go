package drawing

import (
	"fmt"
	"image/color"
	"log/slog"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Generates a color gradient between the given start and end colors.
//
// Parameters:
//
//	start: the starting color
//	end: the ending color
//	numSteps: the number of steps in the gradient
//
// Returns:
//
//	[]lipgloss.Color: the array of colors representing the gradient
func generateColorGradient(start, end lipgloss.Color, numSteps int) []lipgloss.Color {
	colorFrom, e := lipglossToRGBA(start)
	if e != nil {
		colorFrom = color.RGBA{255, 255, 255, 255}
		slog.Error("error converting lipgloss color to RGBA", "error", e)
	}

	colorTo, e := lipglossToRGBA(end)
	if e != nil {
		colorTo = color.RGBA{255, 255, 255, 255}
		slog.Error("error converting lipgloss color to RGBA", "error", e)
	}

	gradient := make([]lipgloss.Color, numSteps)
	rStep := (float64(colorTo.R) - float64(colorFrom.R)) / float64(numSteps-1)
	gStep := (float64(colorTo.G) - float64(colorFrom.G)) / float64(numSteps-1)
	bStep := (float64(colorTo.B) - float64(colorFrom.B)) / float64(numSteps-1)
	aStep := (float64(colorTo.A) - float64(colorFrom.A)) / float64(numSteps-1)

	for i := 0; i < numSteps; i++ {
		r := uint8(float64(colorFrom.R) + float64(i)*rStep)
		g := uint8(float64(colorFrom.G) + float64(i)*gStep)
		b := uint8(float64(colorFrom.B) + float64(i)*bStep)
		a := uint8(float64(colorFrom.A) + float64(i)*aStep)
		gradient[i] = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, a))
	}

	return gradient
}

// Converts a lipgloss color to a color.RGBA.
func lipglossToRGBA(lipglossColor lipgloss.Color) (color.RGBA, error) {
	hex := string(lipglossColor)

	if !isHexColor(hex) {
		return color.RGBA{}, fmt.Errorf("invalid color: %s", hex)
	}

	hex = strings.TrimPrefix(hex, "#")

	val, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.RGBA{}, err
	}

	switch len(hex) {
	case 8:
		return color.RGBA{
			R: uint8(val >> 24 & 0xFF),
			G: uint8(val >> 16 & 0xFF),
			B: uint8(val >> 8 & 0xFF),
			A: uint8(val & 0xFF),
		}, nil
	default:
		return color.RGBA{
			R: uint8(val >> 16 & 0xFF),
			G: uint8(val >> 8 & 0xFF),
			B: uint8(val & 0xFF),
			A: 225,
		}, nil
	}
}

// Checks if the given string is a valid hexadecimal color code(#RRGGBB or #RRGGBBAA).
func isHexColor(c string) bool {
	if (len(c) != 7 && len(c) != 9) || !strings.HasPrefix(c, "#") {
		return false
	}
	c = c[1:]
	for _, char := range c {
		if !(char >= '0' && char <= '9') && !(char >= 'A' && char <= 'F') && !(char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
