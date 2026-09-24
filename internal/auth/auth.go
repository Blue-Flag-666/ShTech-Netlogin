package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Blue-Flag-666/ShTech-Netlogin/internal/config"
)

const maxResponseBytes = 1 << 20

type State int

const (
	Offline State = iota
	Online
	Captive
)

func (s State) String() string {
	switch s {
	case Online:
		return "online"
	case Captive:
		return "captive"
	default:
		return "offline"
	}
}

type PortalParams struct {
	PushPageID string
	SSID       string
	IPAddress  string
	ACIP       string
	UMAC       string
}

type OCR interface {
	Recognize([]byte) (string, error)
}

type Authenticator struct {
	cfg    config.Config
	client *http.Client
	ocr    OCR
}

type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

func New(cfg config.Config, ocr OCR) (*Authenticator, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: cfg.Insecure} //nolint:gosec -- explicit opt-in compatibility switch
	return &Authenticator{
		cfg:    cfg,
		ocr:    ocr,
		client: &http.Client{Timeout: cfg.Timeout, Jar: jar, Transport: transport},
	}, nil
}

func (a *Authenticator) Check(ctx context.Context) (State, PortalParams, error) {
	var selectedAddresses []string
	explicitSelection := a.cfg.IPAddress != "" || a.cfg.Interface != ""
	if explicitSelection {
		var err error
		selectedAddresses, _, err = a.campusIPv4Addresses()
		if err != nil {
			return Offline, PortalParams{}, err
		}
	}
	var errs []error
	for _, probe := range a.cfg.Probes {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probe.URL, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		resp, err := a.client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", probe.URL, err))
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if readErr != nil {
			errs = append(errs, readErr)
			continue
		}

		if params, ok := parsePortalURL(resp.Request.URL); ok {
			if explicitSelection {
				params.IPAddress = selectRedirectAddress(params.IPAddress, selectedAddresses)
			}
			return Captive, params, nil
		}
		if resp.StatusCode == probe.Status && (probe.Contains == "" || strings.Contains(string(body), probe.Contains)) {
			return Online, PortalParams{}, nil
		}
		errs = append(errs, fmt.Errorf("%s 返回 HTTP %d，最终地址 %s", probe.URL, resp.StatusCode, resp.Request.URL.Redacted()))
	}

	params, err := a.discoverDirect(ctx)
	if err == nil {
		return Captive, params, nil
	}
	errs = append(errs, err)
	return Offline, PortalParams{}, errors.Join(errs...)
}

func parsePortalURL(u *url.URL) (PortalParams, bool) {
	q := u.Query()
	params := PortalParams{
		PushPageID: q.Get("pushPageId"), SSID: q.Get("ssid"),
		IPAddress: q.Get("uaddress"), ACIP: q.Get("ac-ip"), UMAC: q.Get("umac"),
	}
	return params, params.PushPageID != "" && params.IPAddress != ""
}

func (a *Authenticator) discoverDirect(ctx context.Context) (PortalParams, error) {
	addresses, _, err := a.campusIPv4Addresses()
	if err != nil {
		return PortalParams{}, err
	}
	var errs []error
	for _, address := range addresses {
		u := a.cfg.BaseURL + "/portal?uaddress=" + url.QueryEscape(address) + "&ac-ip=0"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := a.client.Do(req)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if params, ok := parsePortalURL(resp.Request.URL); ok {
			return params, nil
		}
	}
	if len(addresses) == 0 {
		return PortalParams{}, errors.New("没有找到校园网 10.x IPv4 地址")
	}
	return PortalParams{}, fmt.Errorf("无法从门户发现认证参数: %w", errors.Join(errs...))
}

func (a *Authenticator) campusIPv4Addresses() ([]string, bool, error) {
	if a.cfg.IPAddress != "" {
		ip := net.ParseIP(a.cfg.IPAddress)
		if !isCampusIPv4(ip) {
			return nil, true, fmt.Errorf("--ip 必须是 10.0.0.0/8 内的 IPv4 地址: %q", a.cfg.IPAddress)
		}
		return []string{ip.String()}, true, nil
	}

	var addrs []net.Addr
	var err error
	if a.cfg.Interface != "" {
		iface, lookupErr := net.InterfaceByName(a.cfg.Interface)
		if lookupErr != nil {
			return nil, true, fmt.Errorf("查找网络接口 %q: %w", a.cfg.Interface, lookupErr)
		}
		if iface.Flags&net.FlagUp == 0 {
			return nil, true, fmt.Errorf("网络接口 %q 未启用", a.cfg.Interface)
		}
		addrs, err = iface.Addrs()
	} else {
		addrs, err = net.InterfaceAddrs()
	}
	if err != nil {
		return nil, a.cfg.Interface != "", err
	}
	var result []string
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil && isCampusIPv4(ip) {
			result = append(result, ip.String())
		}
	}
	sort.Strings(result)
	if a.cfg.Interface != "" && len(result) == 0 {
		return nil, true, fmt.Errorf("网络接口 %q 没有 10.0.0.0/8 IPv4 地址", a.cfg.Interface)
	}
	return result, a.cfg.Interface != "", nil
}

func isCampusIPv4(ip net.IP) bool {
	ipv4 := ip.To4()
	return ipv4 != nil && ipv4[0] == 10
}

func selectRedirectAddress(redirected string, selected []string) string {
	for _, address := range selected {
		if address == redirected {
			return address
		}
	}
	return selected[0]
}

func (a *Authenticator) Login(ctx context.Context, params PortalParams) error {
	if a.cfg.FastLogin {
		result, err := a.submitFast(ctx, params)
		if err == nil {
			if result.Success {
				return nil
			}
			if err := permanentLoginError(result); err != nil {
				return err
			}
		}
	}

	if a.ocr == nil {
		return errors.New("验证码识别器未初始化")
	}
	for attempt := 1; attempt <= a.cfg.MaxCaptchaAttempts; attempt++ {
		image, err := a.fetchCaptcha(ctx, params)
		if err != nil {
			return err
		}
		code, err := a.ocr.Recognize(image)
		if err != nil {
			return fmt.Errorf("识别验证码: %w", err)
		}
		if code == "" {
			continue
		}

		result, err := a.submit(ctx, params, code)
		if err != nil {
			return err
		}
		if result.Success {
			return nil
		}
		switch result.ErrorCode {
		case "3010":
			continue
		default:
			if err := permanentLoginError(result); err != nil {
				return err
			}
			message := result.Message
			if message == "" {
				message = "门户返回未知错误"
			}
			return fmt.Errorf("%s (errorcode=%s)", message, result.ErrorCode)
		}
	}
	return fmt.Errorf("验证码连续 %d 次识别失败", a.cfg.MaxCaptchaAttempts)
}

func permanentLoginError(result authResult) error {
	switch result.ErrorCode {
	case "10505":
		return &PermanentError{fmt.Errorf("账号已锁定，剩余 %s 分钟", stringField(result.Data, "remainLockTime"))}
	case "10503":
		if result.Data == nil {
			return &PermanentError{errors.New("账号不存在")}
		}
		return &PermanentError{fmt.Errorf("密码错误；再错 %s 次将锁定 %s 分钟", stringField(result.Data, "remainTimes"), stringField(result.Data, "lockTime"))}
	default:
		return nil
	}
}

func (a *Authenticator) fetchCaptcha(ctx context.Context, params PortalParams) ([]byte, error) {
	q := url.Values{"uaddress": {params.IPAddress}, "date": {strconv.FormatInt(time.Now().UnixMilli(), 10)}}
	if params.ACIP != "" {
		q.Set("acip", params.ACIP)
	}
	if params.UMAC != "" {
		q.Set("umac", params.UMAC)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.BaseURL+"/portalauth/verificationcode?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取验证码: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取验证码: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
}

type authResult struct {
	Success   bool           `json:"success"`
	ErrorCode string         `json:"errorcode"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data"`
}

func (a *Authenticator) submitFast(ctx context.Context, p PortalParams) (authResult, error) {
	form := url.Values{
		"userName": {a.cfg.Username},
		"userPass": {a.cfg.Password},
		"authType": {"1"},
		"uaddress": {p.IPAddress},
		"agreed":   {"1"},
	}
	return a.submitForm(ctx, form)
}

func (a *Authenticator) submit(ctx context.Context, p PortalParams, code string) (authResult, error) {
	form := url.Values{
		"pushPageId": {p.PushPageID}, "userPass": {a.cfg.Password}, "authType": {"1"},
		"ssid": {p.SSID}, "uaddress": {p.IPAddress}, "umac": {p.UMAC}, "acip": {p.ACIP},
		"agreed": {"1"}, "validCode": {code}, "userName": {a.cfg.Username},
		"esn": {""}, "apmac": {""}, "armac": {""}, "accessMac": {""},
		"businessType": {""}, "registerCode": {""}, "questions": {""},
		"dynamicValidCode": {""}, "dynamicRSAToken": {""},
	}
	return a.submitForm(ctx, form)
}

func (a *Authenticator) submitForm(ctx context.Context, form url.Values) (authResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/portalauth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return authResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := a.client.Do(req)
	if err != nil {
		return authResult{}, fmt.Errorf("提交登录: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return authResult{}, fmt.Errorf("提交登录: HTTP %d", resp.StatusCode)
	}
	var result authResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return authResult{}, fmt.Errorf("解析登录响应: %w", err)
	}
	return result, nil
}

func stringField(data map[string]any, name string) string {
	if value, ok := data[name]; ok {
		return fmt.Sprint(value)
	}
	return "?"
}
