package binmanager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/iamnotpirates/idlixdownloader/internal/config"
)

type BinPaths struct {
	FFmpeg    string
	NM3U8DLRE string
}

type BinManager struct {
	binDir string
}

func New() *BinManager {
	return &BinManager{
		binDir: config.GetBinDir(),
	}
}

func (bm *BinManager) GetNM3U8DLREPath() string {
	exeName := "N_m3u8DL-RE"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}

	// 1. Check local AppData bin dir
	appDataPath := filepath.Join(bm.binDir, exeName)
	if fileExists(appDataPath) {
		return appDataPath
	}

	// 2. Check D:\Tools\idlixdownloader\bin
	toolsPath := filepath.Join("D:\\Tools\\idlixdownloader\\bin", exeName)
	if fileExists(toolsPath) {
		return toolsPath
	}

	// 3. Check system PATH
	if p, err := exec.LookPath(exeName); err == nil {
		return p
	}

	return ""
}

func (bm *BinManager) GetFFmpegPath() string {
	exeName := "ffmpeg"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}

	// 1. Check local AppData bin dir
	appDataPath := filepath.Join(bm.binDir, exeName)
	if fileExists(appDataPath) {
		return appDataPath
	}

	// 2. Check D:\Tools\idlixdownloader\bin
	toolsPath := filepath.Join("D:\\Tools\\idlixdownloader\\bin", exeName)
	if fileExists(toolsPath) {
		return toolsPath
	}

	// 3. Check system PATH
	if p, err := exec.LookPath(exeName); err == nil {
		return p
	}

	return ""
}

func (bm *BinManager) EnsureBinaries() (*BinPaths, error) {
	_ = os.MkdirAll(bm.binDir, 0755)

	nPath := bm.GetNM3U8DLREPath()
	if nPath == "" {
		fmt.Println("[BinManager] N_m3u8DL-RE not found. Auto-downloading...")
		var err error
		nPath, err = bm.downloadNM3U8DLRE()
		if err != nil {
			return nil, fmt.Errorf("failed to download N_m3u8DL-RE: %w", err)
		}
	} else if filepath.Dir(nPath) != bm.binDir {
		// Copy to AppData bin dir if loaded from external location
		target := filepath.Join(bm.binDir, filepath.Base(nPath))
		if !fileExists(target) {
			_ = copyBinary(nPath, target)
			nPath = target
		}
	}

	fPath := bm.GetFFmpegPath()
	if fPath == "" {
		fmt.Println("[BinManager] ffmpeg not found. Auto-downloading...")
		var err error
		fPath, err = bm.downloadFFmpeg()
		if err != nil {
			return nil, fmt.Errorf("failed to download ffmpeg: %w", err)
		}
	} else if filepath.Dir(fPath) != bm.binDir {
		// Copy to AppData bin dir if loaded from external location
		target := filepath.Join(bm.binDir, filepath.Base(fPath))
		if !fileExists(target) {
			_ = copyBinary(fPath, target)
			fPath = target
		}
	}

	return &BinPaths{
		FFmpeg:    fPath,
		NM3U8DLRE: nPath,
	}, nil
}

func copyBinary(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func (bm *BinManager) downloadNM3U8DLRE() (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/nilaoda/N_m3u8DL-RE/releases/latest", nil)
	req.Header.Set("User-Agent", "IDLIXDownloader-Go/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	var downloadURL string
	for _, a := range release.Assets {
		if strings.Contains(strings.ToLower(a.Name), "win-x64") && strings.HasSuffix(a.Name, ".zip") {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", fmt.Errorf("compatible N_m3u8DL-RE release asset not found")
	}

	zipPath := filepath.Join(bm.binDir, "nm3u8dl.zip")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return "", err
	}
	defer os.Remove(zipPath)

	if err := extractZipBinary(zipPath, "N_m3u8DL-RE.exe", filepath.Join(bm.binDir, "N_m3u8DL-RE.exe")); err != nil {
		return "", err
	}

	return filepath.Join(bm.binDir, "N_m3u8DL-RE.exe"), nil
}

func (bm *BinManager) downloadFFmpeg() (string, error) {
	downloadURL := "https://github.com/GyanD/codexffmpeg/releases/download/7.1/ffmpeg-7.1-essentials_build.zip"
	zipPath := filepath.Join(bm.binDir, "ffmpeg.zip")

	if err := downloadFile(downloadURL, zipPath); err != nil {
		return "", err
	}
	defer os.Remove(zipPath)

	if err := extractZipBinary(zipPath, "ffmpeg.exe", filepath.Join(bm.binDir, "ffmpeg.exe")); err != nil {
		return "", err
	}

	return filepath.Join(bm.binDir, "ffmpeg.exe"), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad HTTP status: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractZipBinary(zipPath, targetName, outPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == targetName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			outFile, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}
	}
	return fmt.Errorf("file %s not found in zip archive", targetName)
}
