package report

import "testing"

func TestFormatSolidTotal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ml   float64
		g    float64
		want string
	}{
		{"both", 60, 25, "60 mL, 25 g"},
		{"ml only", 60, 0, "60 mL"},
		{"grams only", 0, 25, "25 g"},
		{"neither", 0, 0, "-"},
		{"rounds each value", 59.6, 24.4, "60 mL, 24 g"},
	}
	for _, tt := range tests {
		got := formatSolidTotal(tt.ml, tt.g)
		if got != tt.want {
			t.Errorf("%s: formatSolidTotal(%v, %v) = %q, want %q", tt.name, tt.ml, tt.g, got, tt.want)
		}
	}
}
