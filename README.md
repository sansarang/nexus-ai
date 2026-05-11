# 뚝딱PC

클릭 한 번으로 PC 문제를 뚝딱 해결하는 Windows 유틸리티

## 기술 스택

| 레이어 | 기술 |
|--------|------|
| Frontend | React 18 + TypeScript + TailwindCSS + Framer Motion |
| Runtime | Tauri 2.0 (Rust) |
| Backend | Go 1.22 (Windows 시스템 API) |
| 상태관리 | Zustand |
| 폰트 | Pretendard |

## 개발 환경 실행 (macOS / Windows)

```bash
npm install
npm run tauri dev
```

> Go 백엔드 없이도 더미 데이터로 UI 개발 가능

## Windows 배포 빌드

```bat
build_windows.bat
```

결과물: `src-tauri/target/release/bundle/nsis/뚝딱PC_1.0.0_x64-setup.exe`

### 사전 요구사항 (Windows 빌드 PC)

- [Node.js 20+](https://nodejs.org)
- [Rust (rustup)](https://rustup.rs) — `rustup target add x86_64-pc-windows-msvc`
- [Go 1.22+](https://go.dev/dl)

## 라이선스 키 테스트

**오프라인 키** (백엔드 없이 동작):
```
DDDD-OFFL-TEST-0001
DDDD-OFFL-TEST-0002
```

**온라인 키 생성** (서버 필요):
```bash
# 미구현 — 추후 서버 연동
```

## 프로젝트 구조

```
뚝딱PC/
├── src/                        # React 프론트엔드
│   ├── components/
│   │   ├── CommandPalette/     # Alt+Space 팔레트
│   │   ├── Dashboard/          # 메인 대시보드
│   │   ├── Onboarding/         # 첫 실행 가이드
│   │   ├── License/            # 라이선스 인증
│   │   └── Settings/           # 설정
│   ├── stores/appStore.ts      # Zustand 전역 상태
│   └── hooks/useCommands.ts    # 명령어 목록
├── src-tauri/                  # Rust/Tauri 백엔드
│   ├── src/main.rs             # 트레이, 단축키, 윈도우
│   ├── tauri.conf.json         # Tauri 설정
│   └── icons/                  # 앱 아이콘
├── backend/                    # Go HTTP 사이드카
│   ├── main.go                 # Windows 전용
│   ├── handlers.go             # Windows 전용
│   └── main_stub.go            # non-Windows 빌드용 stub
└── build_windows.bat           # Windows 원클릭 빌드
```

## 단축키

| 단축키 | 동작 |
|--------|------|
| `Alt+Space` | 명령 팔레트 열기/닫기 |
| `↑↓` | 항목 탐색 |
| `Enter` | 실행 |
| `Esc` | 닫기 |
| `Ctrl+,` | 설정 |
