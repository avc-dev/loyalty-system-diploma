package luhn

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "Valid number 79927398713",
			number: "79927398713",
			want:   true,
		},
		{
			name:   "Valid number 12345678903",
			number: "12345678903",
			want:   true,
		},
		{
			name:   "Valid number 4561261212345467",
			number: "4561261212345467",
			want:   true,
		},
		{
			name:   "Valid number with spaces",
			number: "4561 2612 1234 5467",
			want:   true,
		},
		{
			name:   "Invalid number 79927398714",
			number: "79927398714",
			want:   false,
		},
		{
			name:   "Invalid number 12345678900",
			number: "12345678900",
			want:   false,
		},
		{
			name:   "Empty string",
			number: "",
			want:   false,
		},
		{
			name:   "String with letters",
			number: "1234567890a",
			want:   false,
		},
		{
			name:   "Single digit 0",
			number: "0",
			want:   true,
		},
		{
			name:   "Single digit 1",
			number: "1",
			want:   false,
		},
		{
			name:   "Valid number 2377225624",
			number: "2377225624",
			want:   true,
		},
		{
			name:   "Valid number 9278923470",
			number: "9278923470",
			want:   true,
		},
		{
			name:   "Valid number 346436439",
			number: "346436439",
			want:   true,
		},
		{
			name:   "Special characters",
			number: "1234-5678-9012",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.number); got != tt.want {
				t.Errorf("Validate(%q) = %v, want %v", tt.number, got, tt.want)
			}
		})
	}
}

func BenchmarkValidate(b *testing.B) {
	validNumbers := []string{
		"79927398713",
		"12345678903",
		"4561261212345467",
		"2377225624",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(validNumbers[i%len(validNumbers)])
	}
}
