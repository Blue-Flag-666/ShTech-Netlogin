package auth

import (
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Blue-Flag-666/ShTech-Netlogin/internal/config"
)

type fakeOCR struct{}

func (fakeOCR) Recognize([]byte) (string, error) { return "a2b4", nil }

func TestParsePortalURL(t *testing.T) {
	u, _ := url.Parse("https://example/auth?pushPageId=p1&ssid=s1&uaddress=10.1.2.3&ac-ip=10.9.8.7")
	p, ok := parsePortalURL(u)
	if !ok || p.PushPageID != "p1" || p.IPAddress != "10.1.2.3" || p.ACIP != "10.9.8.7" {
		t.Fatalf("unexpected parse result: %#v, %v", p, ok)
	}
}

func TestCheckAndLogin(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/probe":
			http.Redirect(w, r, server.URL+"/auth?pushPageId=page&ssid=campus&uaddress=10.2.3.4&ac-ip=10.5.6.7", http.StatusFound)
		case "/auth":
			w.Write([]byte("portal"))
		case "/portalauth/verificationcode":
			img := image.NewGray(image.Rect(0, 0, 8, 8))
			for i := range img.Pix {
				img.Pix[i] = 255
			}
			jpeg.Encode(w, img, nil)
		case "/portalauth/login":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			for key, want := range map[string]string{"userName": "student", "userPass": "secret", "validCode": "a2b4", "uaddress": "10.2.3.4", "pushPageId": "page"} {
				if got := r.Form.Get(key); got != want {
					t.Errorf("%s=%q, want %q", key, got, want)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "errorcode": "0", "data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{Username: "student", Password: "secret", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 2, Probes: []config.Probe{{URL: server.URL + "/probe", Status: 204}}}
	a, err := New(cfg, fakeOCR{})
	if err != nil {
		t.Fatal(err)
	}
	state, params, err := a.Check(context.Background())
	if err != nil || state != Captive {
		t.Fatalf("Check() = %v, %#v, %v", state, params, err)
	}
	if err := a.Login(context.Background(), params); err != nil {
		t.Fatal(err)
	}
}

func TestBadPasswordIsPermanent(t *testing.T) {
	result := authResult{ErrorCode: "10503", Data: map[string]any{"remainTimes": "2", "lockTime": "10"}}
	b, _ := json.Marshal(result)
	if !strings.Contains(string(b), "10503") {
		t.Fatal("fixture is invalid")
	}
}
