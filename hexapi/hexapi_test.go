// Test cases for hexapi stuff

package hexapi

import "testing"

func TestValuesDiffer(t *testing.T) {
	for _, c := range []struct {
		a, b int
		want bool
	}{
		{1, 1, false},
		{1, 2, true},
		{0, -0, false},
		{-1, 1, true},
	} {
		// a := 1
		// b := 1
		got := valuesDiffer(c.a, c.b)
		if got != c.want {
			t.Errorf("valuesDiffer(%q, %q) == %v but we expected %v", c.a, c.b, got, c.want)
		}
	}
}

func TestFloatToInt(t *testing.T) {
	for _, c := range []struct {
		a    float64
		want int
	}{
		{1.0, 1},
		{2.0, 2},
		{-0.0, 0},
		{-1.0, -1},
	} {
		got := floatToInt(c.a)
		if got != c.want {
			t.Errorf("floatToInt(%v) == %v but we expected %v", c.a, got, c.want)
		}
	}
}
