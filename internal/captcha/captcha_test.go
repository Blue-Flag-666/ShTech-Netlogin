package captcha

import "testing"

func TestDecodeCTC(t *testing.T) {
	r := &Recognizer{charset: []string{" ", "a", "b", "2"}}
	got, err := r.decodeCTC([]int64{0, 1, 1, 0, 2, 2, 3, 0})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ab2" {
		t.Fatalf("got %q", got)
	}
}

func TestModelLoads(t *testing.T) {
	if _, err := New(); err != nil {
		t.Fatal(err)
	}
}
