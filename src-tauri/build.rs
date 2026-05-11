fn main() {
    // Windows 빌드 시에만 Go 사이드카 바이너리 경로 감시
    if std::env::var("CARGO_CFG_TARGET_OS").as_deref() == Ok("windows") {
        println!("cargo:rerun-if-changed=../backend/");
        let sidecar = std::path::Path::new("binaries/backend-x86_64-pc-windows-msvc.exe");
        if !sidecar.exists() {
            println!("cargo:warning=Go sidecar not found at src-tauri/binaries/. Run build_windows.bat first.");
        }
    }
    tauri_build::build()
}
