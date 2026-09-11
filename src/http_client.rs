use serde_json::Value;
use std::path::PathBuf;
use std::process::Stdio;
use tokio::process::Command;

#[derive(Clone)]
pub struct CurlClient {
    pub doh_url: String,
    pub user_agent: String,
    pub cookie_file: PathBuf,
}

impl Default for CurlClient {
    fn default() -> Self {
        Self::new()
    }
}

impl CurlClient {
    pub fn new() -> Self {
        let cookie_file = std::env::temp_dir().join("idlix_curl_cookies.txt");
        Self {
            doh_url: "https://1.1.1.1/dns-query".to_string(),
            user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36".to_string(),
            cookie_file,
        }
    }

    pub async fn get(&self, url: &str, referer: Option<&str>, accept_json: bool) -> Result<String, String> {
        let exe = if cfg!(windows) { "curl.exe" } else { "curl" };
        let cookie_str = self.cookie_file.to_string_lossy();
        let mut cmd = Command::new(exe);
        cmd.args(["-sS", "-L", "--connect-timeout", "15", "--max-time", "45", "--doh-url", &self.doh_url])
            .args(["-c", &cookie_str, "-b", &cookie_str])
            .args(["-H", &format!("User-Agent: {}", self.user_agent)])
            .args(["-H", "Accept-Language: en-US,en;q=0.9,id;q=0.8"]);

        if let Some(ref_url) = referer {
            cmd.args(["-H", &format!("Referer: {ref_url}")]);
            if let Ok(parsed) = url::Url::parse(ref_url) {
                let origin = parsed.origin().ascii_serialization();
                cmd.args(["-H", &format!("Origin: {origin}")]);
            }
        }

        if accept_json {
            cmd.args(["-H", "Accept: application/json"]);
        } else {
            cmd.args(["-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"]);
        }

        cmd.arg(url);
        cmd.stdout(Stdio::piped()).stderr(Stdio::piped());

        #[cfg(windows)]
        cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW

        let output = cmd.output().await.map_err(|e| format!("Failed to execute curl: {e}"))?;
        if !output.status.success() {
            let err = String::from_utf8_lossy(&output.stderr);
            return Err(format!("curl GET failed (exit {}): {}", output.status.code().unwrap_or(-1), err.trim()));
        }

        let body = String::from_utf8_lossy(&output.stdout).to_string();
        Ok(body)
    }

    pub async fn post_json(&self, url: &str, referer: Option<&str>, payload: &Value) -> Result<String, String> {
        let exe = if cfg!(windows) { "curl.exe" } else { "curl" };
        let cookie_str = self.cookie_file.to_string_lossy();
        let json_str = payload.to_string();
        let mut cmd = Command::new(exe);
        cmd.args(["-sS", "-L", "--connect-timeout", "15", "--max-time", "45", "--doh-url", &self.doh_url])
            .args(["-c", &cookie_str, "-b", &cookie_str])
            .args(["-H", &format!("User-Agent: {}", self.user_agent)])
            .args(["-H", "Content-Type: application/json"])
            .args(["-H", "Accept: application/json"])
            .args(["-H", "Accept-Language: en-US,en;q=0.9,id;q=0.8"]);

        if let Some(ref_url) = referer {
            cmd.args(["-H", &format!("Referer: {ref_url}")]);
            if let Ok(parsed) = url::Url::parse(ref_url) {
                let origin = parsed.origin().ascii_serialization();
                cmd.args(["-H", &format!("Origin: {origin}")]);
            }
        }

        cmd.args(["-d", &json_str]);
        cmd.arg(url);
        cmd.stdout(Stdio::piped()).stderr(Stdio::piped());

        #[cfg(windows)]
        cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW

        let output = cmd.output().await.map_err(|e| format!("Failed to execute curl: {e}"))?;
        if !output.status.success() {
            let err = String::from_utf8_lossy(&output.stderr);
            return Err(format!("curl POST failed (exit {}): {}", output.status.code().unwrap_or(-1), err.trim()));
        }

        let body = String::from_utf8_lossy(&output.stdout).to_string();
        Ok(body)
    }
}
