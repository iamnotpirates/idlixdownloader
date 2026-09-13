pub fn open_folder_path(dir_str: &str) {
    let p = std::path::Path::new(dir_str);
    let abs_path = if p.is_absolute() {
        p.to_path_buf()
    } else {
        std::env::current_dir().unwrap_or_default().join(p)
    };

    // Ensure folder exists before opening
    let _ = std::fs::create_dir_all(&abs_path);

    #[cfg(windows)]
    {
        let _ = std::process::Command::new("explorer")
            .arg(&abs_path)
            .spawn();
    }
    #[cfg(not(windows))]
    {
        let _ = open::that(&abs_path);
    }
}

#[cfg(windows)]
const AUTOSTART_REG_KEY: &str = r"SOFTWARE\Microsoft\Windows\CurrentVersion\Run";
#[cfg(windows)]
const AUTOSTART_REG_NAME: &str = "IDLIXDownloader";

#[cfg(windows)]
fn is_autostart_enabled() -> bool {
    use winreg::enums::HKEY_CURRENT_USER;
    use winreg::RegKey;

    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let Ok(run_key) = hkcu.open_subkey(AUTOSTART_REG_KEY) else {
        return false;
    };
    run_key.get_value::<String, _>(AUTOSTART_REG_NAME).is_ok()
}

#[cfg(windows)]
fn set_autostart(enabled: bool) {
    use winreg::enums::{HKEY_CURRENT_USER, KEY_SET_VALUE};
    use winreg::RegKey;

    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let Ok(run_key) = hkcu.open_subkey_with_flags(AUTOSTART_REG_KEY, KEY_SET_VALUE) else {
        return;
    };

    if enabled {
        // Get current exe path, add --silent so startup launch skips browser open
        if let Ok(exe) = std::env::current_exe() {
            let val = format!("\"{}\" --silent", exe.display());
            let _ = run_key.set_value(AUTOSTART_REG_NAME, &val);
        }
    } else {
        let _ = run_key.delete_value(AUTOSTART_REG_NAME);
    }
}

#[cfg(windows)]
pub fn run_tray_loop(server_url: String, silent: bool) -> Result<(), Box<dyn std::error::Error>> {
    use image::GenericImageView;
    use tray_icon::{
        menu::{CheckMenuItem, Menu, MenuEvent, MenuItem, PredefinedMenuItem},
        Icon, TrayIconBuilder, TrayIconEvent,
    };
    use windows_sys::Win32::UI::WindowsAndMessaging::{
        DispatchMessageW, GetMessageW, TranslateMessage, MSG,
    };

    // Load embedded icon
    let icon_bytes = include_bytes!("../assets/icon.png");
    let img = image::load_from_memory(icon_bytes)?;
    let (width, height) = img.dimensions();
    let rgba = img.into_rgba8().into_raw();
    let icon = Icon::from_rgba(rgba, width, height)?;

    let menu = Menu::new();
    let item_open = MenuItem::new("🌐 Open IDLIX Downloader", true, None);
    let item_movies = MenuItem::new("🎬 Open Movies Folder", true, None);
    let item_series = MenuItem::new("📺 Open TV Series Folder", true, None);
    let item_sep1 = PredefinedMenuItem::separator();
    let item_autostart = CheckMenuItem::new("🚀 Start with Windows", true, is_autostart_enabled(), None);
    let item_sep2 = PredefinedMenuItem::separator();
    let item_exit = MenuItem::new("❌ Exit", true, None);

    menu.append(&item_open)?;
    menu.append(&item_movies)?;
    menu.append(&item_series)?;
    menu.append(&item_sep1)?;
    menu.append(&item_autostart)?;
    menu.append(&item_sep2)?;
    menu.append(&item_exit)?;

    let open_id = item_open.id().clone();
    let movies_id = item_movies.id().clone();
    let series_id = item_series.id().clone();
    let autostart_id = item_autostart.id().clone();
    let exit_id = item_exit.id().clone();

    let _tray_icon = TrayIconBuilder::new()
        .with_menu(Box::new(menu))
        .with_menu_on_left_click(false)
        .with_tooltip("IDLIX Downloader")
        .with_icon(icon)
        .build()?;

    // Only open browser on startup if NOT in silent mode
    if !silent {
        let _ = open::that(&server_url);
    }

    let menu_channel = MenuEvent::receiver();
    let tray_channel = TrayIconEvent::receiver();

    unsafe {
        let mut msg: MSG = std::mem::zeroed();
        while GetMessageW(&mut msg, std::ptr::null_mut(), 0, 0) > 0 {
            TranslateMessage(&msg);
            DispatchMessageW(&msg);

            // Process menu events
            while let Ok(event) = menu_channel.try_recv() {
                if event.id == open_id {
                    let _ = open::that(&server_url);
                } else if event.id == movies_id {
                    let cfg = crate::models::AppConfig::load();
                    open_folder_path(&cfg.movies_dir);
                } else if event.id == series_id {
                    let cfg = crate::models::AppConfig::load();
                    open_folder_path(&cfg.series_dir);
                } else if event.id == autostart_id {
                    let new_state = !is_autostart_enabled();
                    set_autostart(new_state);
                    item_autostart.set_checked(new_state);
                } else if event.id == exit_id {
                    std::process::exit(0);
                }
            }

            // Process tray icon clicks
            while let Ok(event) = tray_channel.try_recv() {
                if let TrayIconEvent::DoubleClick { .. } = event {
                    let _ = open::that(&server_url);
                }
            }
        }
    }

    Ok(())
}

#[cfg(not(windows))]
pub fn run_tray_loop(server_url: String, silent: bool) -> Result<(), Box<dyn std::error::Error>> {
    if !silent {
        let _ = open::that(&server_url);
    }
    std::thread::park();
    Ok(())
}
