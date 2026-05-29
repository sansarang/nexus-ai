fn main() {
    // Windows 빌드 시에만 Go 사이드카 바이너리 경로 감시
    if std::env::var("CARGO_CFG_TARGET_OS").as_deref() == Ok("windows") {
        println!("cargo:rerun-if-changed=../backend/");
        let target = std::env::var("TARGET").unwrap_or_default();
        let sidecar = std::path::PathBuf::from(format!("binaries/backend-{}.exe", target));
        if !sidecar.exists() {
            println!("cargo:warning=Go sidecar not found at src-tauri/binaries/. Run build-on-windows.ps1 first.");
        }
        // Resource files
        println!("cargo:rerun-if-changed=backend-bin/nexus-backend.exe");
        println!("cargo:rerun-if-changed=backend-bin/nexus-python.exe");
    }
    tauri_build::build()
}
