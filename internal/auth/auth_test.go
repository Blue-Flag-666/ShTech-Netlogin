package auth

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"net/url"
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
			for key, want := range map[string]string{"userName": "student", "userPass": "secret", "validCode": "a2b4", "uaddress": "10.9.8.7", "pushPageId": "page"} {
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

	cfg := config.Config{Username: "student", Password: "secret", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 2, IPAddress: "10.9.8.7", Probes: []config.Probe{{URL: server.URL + "/probe", Status: 204}}}
	a, err := New(cfg, fakeOCR{})
	if err != nil {
		t.Fatal(err)
	}
	state, params, err := a.Check(context.Background())
	if err != nil || state != Captive {
		t.Fatalf("Check() = %v, %#v, %v", state, params, err)
	}
	if params.IPAddress != "10.9.8.7" {
		t.Fatalf("explicit IP was not applied: %#v", params)
	}
	if err := a.Login(context.Background(), params); err != nil {
		t.Fatal(err)
	}
}

func TestLoginErrorControlFlow(t *testing.T) {
	tests := []struct {
		name      string
		responses []authResult
		permanent bool
		wantErr   bool
		wantCalls int
	}{
		{name: "bad password", responses: []authResult{{ErrorCode: "10503", Data: map[string]any{"remainTimes": "2", "lockTime": "10"}}}, permanent: true, wantErr: true, wantCalls: 1},
		{name: "locked account", responses: []authResult{{ErrorCode: "10505", Data: map[string]any{"remainLockTime": "9"}}}, permanent: true, wantErr: true, wantCalls: 1},
		{name: "captcha retry", responses: []authResult{{ErrorCode: "3010"}, {Success: true}}, wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/portalauth/verificationcode":
					w.Write([]byte("captcha"))
				case "/portalauth/login":
					response := tt.responses[calls]
					calls++
					json.NewEncoder(w).Encode(response)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := config.Config{Username: "student", Password: "wrong", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 3}
			a, err := New(cfg, fakeOCR{})
			if err != nil {
				t.Fatal(err)
			}
			err = a.Login(context.Background(), PortalParams{IPAddress: "10.2.3.4"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Login() error = %v, want error %v", err, tt.wantErr)
			}
			var permanent *PermanentError
			if errors.As(err, &permanent) != tt.permanent {
				t.Fatalf("PermanentError = %v, want %v (error: %v)", permanent != nil, tt.permanent, err)
			}
			if calls != tt.wantCalls {
				t.Fatalf("login calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestFastLoginSuccess(t *testing.T) {
	loginCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/portalauth/login" {
			t.Errorf("unexpected request to %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		loginCalls++
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for key, want := range map[string]string{"userName": "student", "userPass": "secret", "authType": "1", "uaddress": "10.2.3.4", "agreed": "1"} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s=%q, want %q", key, got, want)
			}
		}
		if r.Form.Has("validCode") || r.Form.Has("pushPageId") {
			t.Errorf("fast login sent captcha-only fields: %v", r.Form)
		}
		json.NewEncoder(w).Encode(authResult{Success: true})
	}))
	defer server.Close()

	cfg := config.Config{Username: "student", Password: "secret", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 3, FastLogin: true}
	a, err := New(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Login(context.Background(), PortalParams{IPAddress: "10.2.3.4"}); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 1 {
		t.Fatalf("login calls = %d, want 1", loginCalls)
	}
}

func TestFastLoginFallsBackToCaptcha(t *testing.T) {
	loginCalls := 0
	captchaCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/portalauth/verificationcode":
			captchaCalls++
			w.Write([]byte("captcha"))
		case "/portalauth/login":
			loginCalls++
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if loginCalls == 1 {
				if r.Form.Has("validCode") {
					t.Error("fast login unexpectedly sent validCode")
				}
				json.NewEncoder(w).Encode(authResult{ErrorCode: "3010"})
				return
			}
			if got := r.Form.Get("validCode"); got != "a2b4" {
				t.Errorf("validCode=%q, want a2b4", got)
			}
			json.NewEncoder(w).Encode(authResult{Success: true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{Username: "student", Password: "secret", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 3, FastLogin: true}
	a, err := New(cfg, fakeOCR{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Login(context.Background(), PortalParams{IPAddress: "10.2.3.4"}); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 2 || captchaCalls != 1 {
		t.Fatalf("login calls = %d, captcha calls = %d; want 2 and 1", loginCalls, captchaCalls)
	}
}

func TestFastLoginPermanentErrorDoesNotFallback(t *testing.T) {
	loginCalls := 0
	captchaCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/portalauth/verificationcode":
			captchaCalls++
		case "/portalauth/login":
			loginCalls++
			json.NewEncoder(w).Encode(authResult{ErrorCode: "10503", Data: map[string]any{"remainTimes": "2", "lockTime": "10"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{Username: "student", Password: "wrong", BaseURL: server.URL, Timeout: time.Second, MaxCaptchaAttempts: 3, FastLogin: true}
	a, err := New(cfg, fakeOCR{})
	if err != nil {
		t.Fatal(err)
	}
	err = a.Login(context.Background(), PortalParams{IPAddress: "10.2.3.4"})
	var permanent *PermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("Login() error = %v, want PermanentError", err)
	}
	if loginCalls != 1 || captchaCalls != 0 {
		t.Fatalf("login calls = %d, captcha calls = %d; want 1 and 0", loginCalls, captchaCalls)
	}
}
