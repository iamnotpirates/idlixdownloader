package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

type CurlClient struct {
	DoHURL     string
	UserAgent  string
	CookieFile string
}

func New() *CurlClient {
	cookieFile := filepath.Join(os.TempDir(), "idlix_go_cookies.txt")
	return &CurlClient{
		DoHURL:     "https://1.1.1.1/dns-query",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		CookieFile: cookieFile,
	}
}

func (c *CurlClient) Get(reqURL string, referer string, acceptJSON bool) (string, error) {
	exe := "curl"
	if runtime.GOOS == "windows" {
		exe = "curl.exe"
	}

	args := []string{
		"-sS", "-L",
		"--connect-timeout", "15",
		"--max-time", "45",
		"--doh-url", c.DoHURL,
		"-c", c.CookieFile,
		"-b", c.CookieFile,
		"-H", "User-Agent: " + c.UserAgent,
		"-H", "Accept-Language: en-US,en;q=0.9,id;q=0.8",
	}

	if referer != "" {
		args = append(args, "-H", "Referer: "+referer)
		if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
			origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			args = append(args, "-H", "Origin: "+origin)
		}
	}

	if acceptJSON {
		args = append(args, "-H", "Accept: application/json")
	} else {
		args = append(args, "-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}

	args = append(args, reqURL)

	cmd := exec.Command(exe, args...)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("curl GET failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

func (c *CurlClient) PostJSON(reqURL string, referer string, payload any) (string, error) {
	exe := "curl"
	if runtime.GOOS == "windows" {
		exe = "curl.exe"
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	args := []string{
		"-sS", "-L",
		"--connect-timeout", "15",
		"--max-time", "45",
		"--doh-url", c.DoHURL,
		"-c", c.CookieFile,
		"-b", c.CookieFile,
		"-H", "User-Agent: " + c.UserAgent,
		"-H", "Content-Type: application/json",
		"-H", "Accept: application/json",
		"-H", "Accept-Language: en-US,en;q=0.9,id;q=0.8",
	}

	if referer != "" {
		args = append(args, "-H", "Referer: "+referer)
		if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
			origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			args = append(args, "-H", "Origin: "+origin)
		}
	}

	args = append(args, "-d", string(jsonBytes), reqURL)

	cmd := exec.Command(exe, args...)
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("curl POST failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}
