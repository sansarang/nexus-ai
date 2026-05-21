#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::{Mutex, OnceLock};
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    App, Emitter, Listener, Manager, Runtime, WebviewWindow,
};
use tauri_plugin_global_shortcut::{Code, GlobalShortcutExt, Modifiers, Shortcut, ShortcutState};

// ═══════════════════════════════════════════════════════════════
// Go 백엔드 바이너리 임베드
// Windows 릴리즈 빌드 시: Go backend bytes가 Nexus.exe 내부에 포함됨
// Mac 개발 빌드 시: 이 코드가 컴파일에서 제외됨 (cfg 조건)
// ═══════════════════════════════════════════════════════════════
#[cfg(target_os = "windows")]
static BACKEND_BYTES: &[u8] = include_bytes!("../backend-bin/nexus-backend.exe");

const APP_VERSION: &str = env!("CARGO_PKG_VERSION");

// 백엔드 프로세스 핸들 (앱 종료 시 kill용)
static BACKEND_PROCESS: OnceLock<Mutex<Option<std::process::Child>>> = OnceLock::new();

// ═══════════════════════════════════════════════════════════════
// 창 제어 헬퍼
// ═══════════════════════════════════════════════════════════════
fn toggle_main_window<R: Runtime>(app: &tauri::AppHandle<R>) {
    if let Some(win) = app.get_webview_window("main") {
        if win.is_visible().unwrap_or(false) {
            let _ = win.hide();
        } else {
            let _ = win.show();
            let _ = win.set_focus();
            let _ = win.center();
        }
    }
}

// ═══════════════════════════════════════════════════════════════
// Tauri 커맨드
// ═══════════════════════════════════════════════════════════════
#[tauri::command]
fn minimize_window(window: WebviewWindow) { let _ = window.minimize(); }

#[tauri::command]
fn toggle_maximize(window: WebviewWindow) {
    if window.is_maximized().unwrap_or(false) { let _ = window.unmaximize(); }
    else { let _ = window.maximize(); }
}

#[tauri::command]
fn close_window(window: WebviewWindow) { let _ = window.hide(); }

#[tauri::command]
fn switch_to_character_mode(app: tauri::AppHandle) {
    if let Some(main) = app.get_webview_window("main") {
        let _ = main.hide();
    }
    if let Some(char_win) = app.get_webview_window("character") {
        if let Ok(Some(m)) = char_win.primary_monitor() {
            let size  = m.size();
            let scale = m.scale_factor();
            let w = 280.0_f64;
            let h = 500.0_f64;
            let x = (size.width  as f64 / scale) - w - 24.0;
            let y = (size.height as f64 / scale) - h - 60.0;
            let _ = char_win.set_position(tauri::PhysicalPosition::new(x as i32, y as i32));
        }
        let _ = char_win.show();
        let _ = char_win.set_focus();
    }
}

#[tauri::command]
fn open_chat_window(app: tauri::AppHandle) {
    if let Some(main) = app.get_webview_window("main") {
        if !main.is_visible().unwrap_or(false) {
            let _ = main.show();
            let _ = main.set_focus();
        }
    }
}

#[tauri::command]
async fn run_diagnostics() -> serde_json::Value { serde_json::json!({ "score": 0, "issues": [] }) }

#[tauri::command]
async fn repair_issue(_id: String) -> serde_json::Value { serde_json::json!({ "success": true }) }

#[tauri::command]
async fn repair_all() -> serde_json::Value { serde_json::json!({ "success": true }) }

// ── 백엔드 준비 상태 확인 (프론트엔드에서 폴링용) ──────────────
#[tauri::command]
async fn check_backend_ready() -> bool {
    match tokio::net::TcpStream::connect("127.0.0.1:17891").await {
        Ok(_) => true,
        Err(_) => false,
    }
}

// ── 설치 후 의존성 상태 조회 (Go 백엔드에 위임) ─────────────────
// 프론트엔드에서 fetch("http://127.0.0.1:17891/api/setup/status")로 직접 호출 가능
// Tauri 커맨드는 백엔드 연결 가능 여부만 확인
#[tauri::command]
async fn get_setup_status() -> serde_json::Value {
    match tokio::net::TcpStream::connect("127.0.0.1:17891").await {
        Ok(_) => serde_json::json!({ "backend_ready": true }),
        Err(_) => serde_json::json!({ "backend_ready": false }),
    }
}

// ── Chrome 경로 탐색 (Windows 전용) ─────────────────────────────
#[cfg(target_os = "windows")]
fn find_chrome_windows() -> Option<String> {
    let paths = vec![
        r"C:\Program Files\Google\Chrome\Application\chrome.exe",
        r"C:\Program Files (x86)\Google\Chrome\Application\chrome.exe",
        r"C:\Program Files\Microsoft\Edge\Application\msedge.exe",
    ];
    for p in &paths {
        if std::path::Path::new(p).exists() {
            return Some(p.to_string());
        }
    }
    // 레지스트리 탐색
    None
}

// ── Outlook 설치 여부 확인 (Windows 전용) ───────────────────────
#[tauri::command]
#[cfg(target_os = "windows")]
async fn check_outlook_installed() -> bool {
    let paths = vec![
        r"C:\Program Files\Microsoft Office\root\Office16\OUTLOOK.EXE",
        r"C:\Program Files (x86)\Microsoft Office\root\Office16\OUTLOOK.EXE",
    ];
    for p in &paths {
        if std::path::Path::new(p).exists() {
            return true;
        }
    }
    // New Outlook (UWP) 확인
    let output = std::process::Command::new("powershell")
        .args(["-WindowStyle", "Hidden", "-Command",
            "if (Get-AppxPackage -Name Microsoft.OutlookForWindows -ErrorAction SilentlyContinue) { 'true' } else { 'false' }"])
        .output();
    matches!(output, Ok(o) if String::from_utf8_lossy(&o.stdout).trim() == "true")
}

#[tauri::command]
#[cfg(not(target_os = "windows"))]
async fn check_outlook_installed() -> bool { false }

// ═══════════════════════════════════════════════════════════════
// Go 백엔드 임베드 + 자동 추출 + 실행
// ═══════════════════════════════════════════════════════════════
// Windows: 콘솔 창 숨김 플래그
#[cfg(target_os = "windows")]
use std::os::windows::process::CommandExt;
const CREATE_NO_WINDOW: u32 = 0x08000000;

#[cfg(target_os = "windows")]
fn get_backend_dir() -> std::path::PathBuf {
    // %APPDATA%\Nexus\ 경로 사용
    let appdata = std::env::var("APPDATA")
        .unwrap_or_else(|_| std::env::temp_dir().to_string_lossy().to_string());
    std::path::PathBuf::from(appdata).join("Nexus")
}

#[cfg(target_os = "windows")]
fn needs_extraction(backend_path: &std::path::Path) -> bool {
    if !backend_path.exists() {
        return true;
    }
    // 버전 파일로 업데이트 여부 판단
    let ver_path = backend_path.parent().unwrap().join("nexus_version.txt");
    match std::fs::read_to_string(&ver_path) {
        Ok(ver) => ver.trim() != APP_VERSION,
        Err(_) => true,
    }
}

#[cfg(target_os = "windows")]
fn extract_backend() -> Result<std::path::PathBuf, String> {
    let dir = get_backend_dir();
    std::fs::create_dir_all(&dir)
        .map_err(|e| format!("AppData 폴더 생성 실패: {e}"))?;

    let backend_path = dir.join("nexus-backend.exe");

    if needs_extraction(&backend_path) {
        std::fs::write(&backend_path, BACKEND_BYTES)
            .map_err(|e| format!("백엔드 추출 실패: {e}"))?;

        // 버전 기록
        let _ = std::fs::write(dir.join("nexus_version.txt"), APP_VERSION);
    }

    Ok(backend_path)
}

fn launch_backend<R: Runtime>(app: &App<R>) {
    // 글로벌 프로세스 핸들 초기화
    BACKEND_PROCESS.get_or_init(|| Mutex::new(None));

    #[cfg(target_os = "windows")]
    {
        // Windows 릴리즈: 임베드된 바이너리 추출 후 실행
        match extract_backend() {
            Ok(path) => {
                match std::process::Command::new(&path)
                    .creation_flags(0x08000000) // CREATE_NO_WINDOW: 콘솔창 숨김
                    .spawn()
                {
                    Ok(child) => {
                        if let Some(mutex) = BACKEND_PROCESS.get() {
                            if let Ok(mut guard) = mutex.lock() {
                                *guard = Some(child);
                            }
                        }
                    }
                    Err(e) => eprintln!("[Nexus] 백엔드 실행 실패: {e}"),
                }
            }
            Err(e) => eprintln!("[Nexus] 백엔드 추출 실패: {e}"),
        }
    }

    // Mac 개발 환경: sidecar 방식 (dev 서버 사용 시)
    #[cfg(not(target_os = "windows"))]
    {
        use tauri_plugin_shell::ShellExt;
        if let Ok(cmd) = app.shell().sidecar("backend") {
            let _ = cmd.spawn();
        }
    }
}

fn kill_backend() {
    if let Some(mutex) = BACKEND_PROCESS.get() {
        if let Ok(mut guard) = mutex.lock() {
            if let Some(child) = guard.as_mut() {
                let _ = child.kill();
            }
            *guard = None;
        }
    }
    // 혹시 살아있는 프로세스 정리 (Windows)
    #[cfg(target_os = "windows")]
    {
        let _ = std::process::Command::new("taskkill")
            .args(["/F", "/IM", "nexus-backend.exe"])
            .output();
    }
}

// ═══════════════════════════════════════════════════════════════
// 시스템 트레이
// ═══════════════════════════════════════════════════════════════
fn setup_tray<R: Runtime>(app: &App<R>) -> tauri::Result<()> {
    let open_item = MenuItem::with_id(app, "open",     "Nexus 열기",    true, None::<&str>)?;
    let sep1      = tauri::menu::PredefinedMenuItem::separator(app)?;
    let quit_item = MenuItem::with_id(app, "quit",     "종료",          true, None::<&str>)?;

    let menu = Menu::with_items(app, &[&open_item, &sep1, &quit_item])?;

    TrayIconBuilder::new()
        .tooltip("Nexus AI 비서 — 클릭하여 열기")
        .menu(&menu)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "open" => toggle_main_window(app),
            "quit" => {
                kill_backend();
                app.exit(0);
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event {
                toggle_main_window(tray.app_handle());
            }
        })
        .build(app)?;

    Ok(())
}

// ═══════════════════════════════════════════════════════════════
// 전역 단축키 Alt+Space
// ═══════════════════════════════════════════════════════════════
fn setup_shortcut<R: Runtime>(app: &App<R>) -> tauri::Result<()> {
    let shortcut = Shortcut::new(Some(Modifiers::ALT), Code::Space);
    let handle   = app.handle().clone();

    app.global_shortcut()
        .on_shortcut(shortcut, move |_, _, event| {
            if event.state == ShortcutState::Pressed {
                if let Some(char_win) = handle.get_webview_window("character") {
                    if char_win.is_visible().unwrap_or(false) {
                        let _ = char_win.emit("wake-word-activated", ());
                        return;
                    }
                }
                if let Some(main) = handle.get_webview_window("main") {
                    if main.is_visible().unwrap_or(false) {
                        let _ = main.emit("toggle-command", ());
                    } else {
                        let _ = main.show();
                        let _ = main.set_focus();
                        let _ = main.center();
                        let _ = main.emit("toggle-command", ());
                    }
                }
            }
        })
        .map_err(|e| tauri::Error::Anyhow(e.into()))?;

    Ok(())
}

// ═══════════════════════════════════════════════════════════════
// main
// ═══════════════════════════════════════════════════════════════
#[tokio::main]
async fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            // 두 번째 실행 시 기존 창을 앞으로 가져옴
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.show();
                let _ = win.set_focus();
                let _ = win.unminimize();
            }
        }))
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_global_shortcut::Builder::new().build())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .setup(|app| {
            // 1. Go 백엔드 실행 (임베드된 바이너리 자동 추출)
            launch_backend(app);

            // 2. 시스템 트레이 설정
            setup_tray(app)?;

            // 3. Alt+Space 단축키 등록
            setup_shortcut(app)?;

            // 4. 온보딩 창 표시 (항상 시작 화면부터)
            if let Some(win) = app.get_webview_window("main") {
                let _ = win.show();
                let _ = win.set_focus();
                let _ = win.center();
            }

            // 5. 딥링크 프로토콜 등록 (nexus://) + OAuth 콜백 처리
            #[cfg(target_os = "windows")]
            {
                use tauri_plugin_deep_link::DeepLinkExt;
                let _ = app.deep_link().register("nexus");
            }
            let handle = app.handle().clone();
            app.listen("deep-link://new-url", move |event| {
                let url = event.payload().to_string();
                // nexus://auth/callback?code=XXX 수신 시 프론트엔드로 전달
                if url.contains("auth/callback") || url.contains("access_token") {
                    if let Some(win) = handle.get_webview_window("main") {
                        let _ = win.show();
                        let _ = win.set_focus();
                        let _ = win.emit("oauth-callback", url);
                    }
                }
            });

            Ok(())
        })
        // 앱 종료 시 백엔드 프로세스 정리
        .on_window_event(|_window, event| {
            if let tauri::WindowEvent::Destroyed = event {
                // 마지막 창 닫힐 때 백엔드 종료
            }
        })
        .invoke_handler(tauri::generate_handler![
            minimize_window,
            toggle_maximize,
            close_window,
            switch_to_character_mode,
            open_chat_window,
            run_diagnostics,
            repair_issue,
            repair_all,
            check_backend_ready,
            get_setup_status,
            check_outlook_installed,
        ])
        .build(tauri::generate_context!())
        .expect("Nexus 실행 실패")
        .run(|app, event| {
            // 앱 종료 직전 백엔드 kill
            if let tauri::RunEvent::Exit = event {
                kill_backend();
            }
            // 모든 창이 닫혀도 트레이에서 계속 실행
            if let tauri::RunEvent::ExitRequested { api, .. } = event {
                api.prevent_exit();
            }
        });
}
