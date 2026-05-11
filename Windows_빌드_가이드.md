# NEXUS AI — Windows 빌드 완전 가이드

## 전체 흐름

```
Mac에서 파일 복사 → Windows에서 빌드 환경 설치 → 한 번 빌드 → Nexus_x64-setup.exe 생성
```

---

## 1단계: Windows PC에 개발 도구 설치

Windows 10/11에서 **PowerShell(관리자)** 로 실행합니다.

### 1-1. Node.js 설치 (프론트엔드 빌드)
https://nodejs.org → **LTS 버전** 다운로드 후 설치  
설치 후 확인:
```powershell
node -v   # v20.x 이상
npm -v    # 10.x 이상
```

### 1-2. Rust 설치 (Tauri 필수)
```powershell
winget install Rustlang.Rustup
# 또는 직접 다운로드: https://rustup.rs
rustup install stable
rustup target add x86_64-pc-windows-msvc
```
설치 후 확인:
```powershell
rustc --version   # rustc 1.77 이상
cargo --version
```

### 1-3. Go 설치 (백엔드 빌드)
https://go.dev/dl/ → **go1.22+ windows-amd64.msi** 다운로드 후 설치  
설치 후 확인:
```powershell
go version   # go1.22 이상
```

### 1-4. Visual Studio Build Tools 설치 (Rust 컴파일러 필수)
```powershell
winget install Microsoft.VisualStudio.2022.BuildTools
```
설치 시 "C++ build tools" 워크로드 선택 필수

### 1-5. WebView2 런타임 (Windows 10 구버전만 필요)
Windows 11은 기본 내장. Windows 10에서 없으면:
https://developer.microsoft.com/microsoft-edge/webview2/ → Evergreen Bootstrapper 다운로드

---

## 2단계: 프로젝트 파일을 Windows로 복사

### Mac에서 압축
```bash
# Mac 터미널에서 실행
cd ~/Desktop
zip -r nexus-source.zip 뚝딱PC \
  --exclude "뚝딱PC/node_modules/*" \
  --exclude "뚝딱PC/dist/*" \
  --exclude "뚝딱PC/src-tauri/target/*" \
  --exclude "뚝딱PC/backend/nexus-backend.exe" \
  --exclude "뚝딱PC/.git/*"
```
생성된 `nexus-source.zip`을 Windows PC로 전송 (USB, Google Drive 등)

### Windows에서 압축 해제
```
C:\Users\사용자이름\Desktop\nexus\  ← 이 경로에 압축 해제
```

---

## 3단계: 의존성 설치

Windows PowerShell에서:
```powershell
cd C:\Users\사용자이름\Desktop\nexus

# 프론트엔드 패키지 설치
npm install

# Go 모듈 다운로드
cd backend
go mod download
cd ..
```

---

## 4단계: 빌드 실행

아래 배치 파일을 실행하면 **자동으로 전체 빌드**됩니다.

```powershell
# 프로젝트 루트에서 실행
.\build_windows.bat
```

빌드 완료 후 생성 파일:
```
src-tauri\target\release\bundle\nsis\Nexus_2.5.0_x64-setup.exe  ← 설치 파일
src-tauri\target\release\bundle\nsis\Nexus_2.5.0_x64_portable.exe  ← 포터블
```

---

## 5단계: 문제 해결

### "NSIS not found" 오류
```powershell
winget install NSIS.NSIS
# 또는: https://nsis.sourceforge.io 에서 직접 설치
```

### "link.exe not found" 오류
Visual Studio Build Tools가 없는 것. 1-4단계 재실행

### Go 빌드 오류 "chromedp requires CGO"
chromedp는 CGO 불필요. 아래로 강제 빌드:
```powershell
cd backend
set CGO_ENABLED=0
go build -o nexus-backend.exe .
```

### WebView2 오류 (실행 시)
Windows 10에서 WebView2 런타임 미설치. 1-5단계 참조

---

## 빠른 테스트 (설치 없이 바로 실행)

빌드 후 설치 파일 대신 포터블 EXE로 먼저 테스트:
```
src-tauri\target\release\Nexus.exe  ← 직접 실행 가능
```
