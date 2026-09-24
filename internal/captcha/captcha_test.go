package captcha

import "testing"

func TestDecodeCTC(t *testing.T) {
	r := &Recognizer{charset: []string{" ", "a", "b", "2"}}
	tests := []struct {
		name   string
		values []int64
		want   string
	}{
		{name: "collapse adjacent repeats", values: []int64{0, 1, 1, 0, 2, 2, 3, 0}, want: "ab2"},
		{name: "blank separates repeats", values: []int64{1, 0, 1}, want: "aa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.decodeCTC(tt.values)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelLoads(t *testing.T) {
	if _, err := New(); err != nil {
		t.Fatal(err)
	}
}
