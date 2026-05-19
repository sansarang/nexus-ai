//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func cmdWorkflowPreset(cx cmdCtx) {
		preset, _ := cx.params["preset"].(string)
		query, _ := cx.params["query"].(string)
		if query == "" {
			query = cx.req.Message
		}
		llmMu.RLock()
		localTKey := llmTavilyKey; cx.tKey = localTKey
		llmMu.RUnlock()

		type wfSection struct{ name, body string }
		wfCh := make(chan wfSection, 4)

		type workflowDef struct {
			title    string
			searches []struct{ name, q string }
			prompt   string
		}

		presetDefs := map[string]workflowDef{
			// ── 개발자 (20개) ───────────────────────────────────────
			"dev_bug_fix": {
				title: "버그 빠르게 해결",
				searches: []struct{ name, q string }{
					{"에러 원인 분석", query + " error fix solution 2025"},
					{"스택오버플로우 해결법", query + " stackoverflow github issue"},
				},
				prompt: `다음 정보를 바탕으로 버그 해결 가이드를 작성해줘.
%s
형식:
1. 에러 원인 분석 (가능성 높은 순)
2. 빠른 진단 체크리스트 (3가지)
3. 단계별 수정 방법
4. 수정 코드 예시 (언어/프레임워크 맞게)
5. 재발 방지 방법`,
			},
			"dev_refactor": {
				title: "코드 리팩토링",
				searches: []struct{ name, q string }{
					{"리팩토링 패턴", "code refactoring patterns best practices 2025"},
					{"클린 코드 원칙", "clean code principles SOLID"},
				},
				prompt: `다음 정보를 바탕으로 코드 리팩토링 가이드를 작성해줘.
%s
형식:
1. 리팩토링이 필요한 코드 냄새(Code Smell) 감지
2. 적용할 리팩토링 패턴 (Extract/Replace/Rename 등)
3. 단계별 리팩토링 순서
4. Before/After 코드 예시
5. 테스트 안전망 구축 방법`,
			},
			"dev_github_search": {
				title: "GitHub 이슈/PR 검색",
				searches: []struct{ name, q string }{
					{"GitHub 이슈 검색", query + " github issue bug fix"},
					{"관련 PR/커밋", query + " github pull request merged solution"},
				},
				prompt: `다음 정보를 바탕으로 GitHub 이슈/PR 검색 결과를 정리해줘.
%s
형식:
1. 관련 이슈 요약 (상태/라벨/우선순위)
2. 연관 PR 목록 및 상태
3. 핵심 해결책 요약
4. 추가로 확인할 저장소/이슈 추천
5. GitHub 검색 팁 (고급 검색 쿼리)`,
			},
			"dev_terminal_command": {
				title: "터미널 명령 최적화",
				searches: []struct{ name, q string }{
					{"터미널 명령어 최적화", query + " terminal command optimization linux mac"},
					{"bash 스크립트 팁", "bash zsh productivity tips 2025"},
				},
				prompt: `다음 정보를 바탕으로 터미널 명령어 최적화 가이드를 작성해줘.
%s
형식:
1. 현재 명령어 분석 및 문제점
2. 최적화된 명령어 제안 (복사 가능)
3. 파이프라인/조합 활용법
4. 알리아스(alias) 등록 예시
5. 추천 터미널 도구 (fzf/ripgrep/bat 등)`,
			},
			"dev_api_design": {
				title: "API 설계",
				searches: []struct{ name, q string }{
					{"REST API 설계 원칙", "REST API design best practices 2025"},
					{"OpenAPI 스펙 예시", "OpenAPI 3.0 specification example"},
				},
				prompt: `다음 정보를 바탕으로 API 설계 가이드를 작성해줘.
%s
형식:
1. 엔드포인트 구조 설계 (RESTful 원칙)
2. 요청/응답 스키마 예시 (JSON)
3. HTTP 메서드 및 상태코드 사용 기준
4. 인증/인가 방식 추천
5. OpenAPI 스펙 초안 (YAML)
6. 버전 관리 전략`,
			},
			"dev_test_generate": {
				title: "테스트 코드 자동 생성",
				searches: []struct{ name, q string }{
					{"단위 테스트 작성법", "unit test best practices 2025"},
					{"테스트 커버리지 전략", "test coverage strategy TDD BDD"},
				},
				prompt: `다음 정보를 바탕으로 테스트 코드 생성 가이드를 작성해줘.
%s
형식:
1. 테스트 전략 (단위/통합/E2E 구분)
2. 테스트 케이스 설계 (Happy/Edge/Error Path)
3. 단위 테스트 코드 예시 (Jest/PyTest/Go test)
4. Mock/Stub 활용 방법
5. 테스트 커버리지 목표 및 측정 방법`,
			},
			"dev_daily_standup": {
				title: "데일리 스탠드업 브리핑",
				searches: []struct{ name, q string }{
					{"스탠드업 미팅 효과적 운영", "daily standup meeting best practice agile"},
				},
				prompt: `다음 정보를 바탕으로 오늘 데일리 스탠드업 브리핑 초안을 작성해줘.
%s
형식:
1. 어제 한 일 (Yesterday)
2. 오늘 할 일 (Today)
3. 블로커/이슈 (Blocker)
4. 팀에 공유할 내용
5. 스탠드업 효과적으로 진행하는 팁`,
			},
			"dev_pr_create": {
				title: "PR 자동 생성",
				searches: []struct{ name, q string }{
					{"PR 작성 베스트 프랙티스", "pull request description best practices"},
					{"PR 템플릿 예시", "github pull request template checklist"},
				},
				prompt: `다음 정보를 바탕으로 PR(Pull Request) 작성 초안을 생성해줘.
%s
형식:
1. PR 제목 (명확하고 간결하게)
2. 변경 사항 요약 (What/Why/How)
3. 테스트 방법
4. 스크린샷/영상 첨부 체크
5. 리뷰어 체크리스트
6. 관련 이슈 링크 형식`,
			},
			"dev_ci_cd": {
				title: "CI/CD 파이프라인 최적화",
				searches: []struct{ name, q string }{
					{"CI/CD 최적화 방법", "CI/CD pipeline optimization 2025"},
					{"GitHub Actions 최적화", "GitHub Actions workflow optimization cache"},
				},
				prompt: `다음 정보를 바탕으로 CI/CD 파이프라인 최적화 가이드를 작성해줘.
%s
형식:
1. 현재 파이프라인 분석 포인트
2. 빌드 속도 최적화 (캐시/병렬화)
3. 테스트 자동화 개선
4. 배포 자동화 전략 (Blue-Green/Canary)
5. 모니터링 및 알림 설정
6. GitHub Actions/GitLab CI 설정 예시`,
			},
			"dev_log_analysis": {
				title: "로그 분석 및 디버깅",
				searches: []struct{ name, q string }{
					{"로그 분석 방법", "log analysis debugging best practice"},
					{"에러 패턴 감지", "error pattern detection log monitoring"},
				},
				prompt: `다음 정보를 바탕으로 로그 분석 및 디버깅 가이드를 작성해줘.
%s
형식:
1. 로그 레벨 분류 (ERROR/WARN/INFO/DEBUG)
2. 이상 패턴 감지 방법
3. 핵심 에러 원인 분석
4. 로그 분석 명령어 (grep/awk/jq)
5. 로그 모니터링 도구 추천 (ELK/Grafana)
6. 알림 설정 방법`,
			},
			"dev_performance": {
				title: "성능 병목점 분석",
				searches: []struct{ name, q string }{
					{"성능 최적화 방법", "application performance optimization 2025"},
					{"프로파일링 도구", "profiling tools performance bottleneck"},
				},
				prompt: `다음 정보를 바탕으로 성능 병목점 분석 및 최적화 가이드를 작성해줘.
%s
형식:
1. 성능 측정 방법 및 지표 (응답시간/처리량/메모리)
2. 프로파일링 도구 사용법
3. 병목점 유형별 원인 (CPU/메모리/I/O/네트워크)
4. 최적화 우선순위 결정 기준
5. 코드 수준 최적화 예시
6. 인프라 수준 개선 방법`,
			},
			"dev_security_scan": {
				title: "보안 취약점 검사",
				searches: []struct{ name, q string }{
					{"OWASP 취약점 2025", "OWASP top 10 vulnerabilities 2025"},
					{"코드 보안 체크리스트", "code security audit checklist dependency vulnerability"},
				},
				prompt: `다음 정보를 바탕으로 보안 취약점 검사 가이드를 작성해줘.
%s
형식:
1. OWASP Top 10 체크 항목
2. 코드 수준 취약점 (인젝션/XSS/CSRF 등)
3. 의존성 취약점 스캔 방법 (npm audit/snyk)
4. 인증/인가 보안 체크포인트
5. 시크릿/API 키 노출 방지
6. 보안 도구 추천 및 자동화`,
			},
			"dev_docker": {
				title: "Docker/K8s 설정",
				searches: []struct{ name, q string }{
					{"Docker 최적화", "Dockerfile optimization best practices 2025"},
					{"Kubernetes 배포", "Kubernetes deployment best practices"},
				},
				prompt: `다음 정보를 바탕으로 Docker/Kubernetes 설정 가이드를 작성해줘.
%s
형식:
1. Dockerfile 최적화 (멀티스테이지 빌드/레이어 최소화)
2. docker-compose.yml 예시
3. Kubernetes Deployment/Service yaml 예시
4. 이미지 크기 줄이는 방법
5. 컨테이너 보안 설정
6. 로컬 개발 환경 구성`,
			},
			"dev_db_optimize": {
				title: "데이터베이스 쿼리 최적화",
				searches: []struct{ name, q string }{
					{"SQL 쿼리 최적화", "SQL query optimization index performance 2025"},
					{"데이터베이스 성능 튜닝", "database performance tuning explain plan"},
				},
				prompt: `다음 정보를 바탕으로 데이터베이스 쿼리 최적화 가이드를 작성해줘.
%s
형식:
1. 느린 쿼리 식별 방법 (EXPLAIN/ANALYZE)
2. 인덱스 설계 전략 (복합/커버링/부분 인덱스)
3. N+1 쿼리 문제 해결
4. 최적화된 쿼리 예시 (Before/After)
5. 캐싱 전략 (Redis/Memcached)
6. 데이터베이스별 특화 팁 (PostgreSQL/MySQL/MongoDB)`,
			},
			"dev_tech_summary": {
				title: "기술 학습 자료 정리",
				searches: []struct{ name, q string }{
					{"기술 공식 문서", query + " official documentation tutorial 2025"},
					{"베스트 프랙티스", query + " best practices examples github"},
				},
				prompt: `다음 정보를 바탕으로 기술 학습 자료를 정리해줘.
%s
형식:
1. 핵심 개념 요약 (3-5가지)
2. 빠른 시작 (Quick Start) 가이드
3. 꼭 알아야 할 API/명령어
4. 추천 학습 순서 및 자료
5. 자주 실수하는 부분 주의사항
6. 실전 예제 코드`,
			},
			"dev_code_style": {
				title: "코드 스타일 일관성 검사",
				searches: []struct{ name, q string }{
					{"코딩 컨벤션 가이드", "coding convention style guide 2025"},
					{"Lint 설정 방법", "ESLint Prettier golangci-lint configuration"},
				},
				prompt: `다음 정보를 바탕으로 코드 스타일 가이드를 작성해줘.
%s
형식:
1. 팀 코딩 컨벤션 핵심 규칙 (네이밍/포맷/구조)
2. Linter 설정 방법 (.eslintrc/.golangci.yml)
3. Prettier/포맷터 설정 예시
4. 자주 발생하는 스타일 위반 패턴
5. pre-commit hook 자동화 설정
6. 코드 리뷰 시 스타일 체크 포인트`,
			},
			"dev_migration": {
				title: "마이그레이션 계획 수립",
				searches: []struct{ name, q string }{
					{"DB 마이그레이션 전략", "database migration strategy zero downtime 2025"},
					{"스키마 변경 방법", "schema migration rollback strategy"},
				},
				prompt: `다음 정보를 바탕으로 마이그레이션 계획을 작성해줘.
%s
형식:
1. 마이그레이션 전 준비사항 (백업/롤백 계획)
2. 단계별 마이그레이션 스크립트 구조
3. Zero-downtime 마이그레이션 전략
4. 데이터 정합성 검증 방법
5. 롤백 시나리오 및 절차
6. 마이그레이션 도구 추천 (Flyway/Liquibase/golang-migrate)`,
			},
			"dev_error_classify": {
				title: "에러 로그 자동 분류",
				searches: []struct{ name, q string }{
					{"에러 분류 체계", "error classification logging strategy"},
					{"에러 모니터링", "error monitoring Sentry Datadog 2025"},
				},
				prompt: `다음 정보를 바탕으로 에러 로그 분류 가이드를 작성해줘.
%s
형식:
1. 에러 카테고리별 분류 기준 (인프라/애플리케이션/외부API/사용자)
2. 심각도 레벨 정의 (Critical/Error/Warning/Info)
3. 에러 코드 체계 설계 방법
4. 자동 분류 규칙 예시 (정규식/패턴)
5. 알림 임계값 설정 방법
6. 에러 대시보드 구성 방법`,
			},
			"dev_weekly_report": {
				title: "주간 개발 리포트",
				searches: []struct{ name, q string }{
					{"개발팀 주간 보고", "engineering weekly report template"},
					{"개발 생산성 지표", "developer productivity metrics DORA"},
				},
				prompt: `다음 정보를 바탕으로 주간 개발 리포트를 작성해줘.
%s
형식:
1. 이번 주 완료한 개발 항목 (기능/버그/리팩토링)
2. PR 현황 (머지됨/리뷰 중/블로킹)
3. 기술 부채 현황
4. 이슈 및 블로커
5. 다음 주 계획
6. DORA 지표 (배포 빈도/변경 리드타임/복구시간)`,
			},
			"dev_code_review": {
				title: "코드 리뷰 준비",
				searches: []struct{ name, q string }{
					{"코드 리뷰 베스트 프랙티스", "code review best practices 2025"},
					{"보안 취약점 체크리스트", "code security checklist OWASP"},
				},
				prompt: `다음 정보를 바탕으로 코드 리뷰 준비 체크리스트를 작성해줘.
%s
형식:
1. 리뷰 전 확인 사항 (5가지)
2. 코드 품질 체크포인트 (가독성/성능/보안)
3. 자주 놓치는 부분
4. 리뷰 코멘트 작성 팁`,
			},
			"dev_deploy_check": {
				title: "배포 체크리스트",
				searches: []struct{ name, q string }{
					{"배포 전 체크리스트", "production deployment checklist 2025"},
					{"장애 대응 롤백", "deployment rollback strategy"},
				},
				prompt: `다음 정보를 바탕으로 배포 체크리스트를 작성해줘.
%s
형식:
1. 배포 전 (코드/테스트/환경변수/DB 마이그레이션)
2. 배포 중 (모니터링 포인트)
3. 배포 후 (헬스체크/로그 확인)
4. 롤백 기준 및 방법`,
			},
			"dev_tech_trend": {
				title: "최신 기술 트렌드",
				searches: []struct{ name, q string }{
					{"2025 개발 트렌드", "software development trends 2025"},
					{"AI 개발 도구 트렌드", "AI developer tools 2025"},
				},
				prompt: `다음 정보를 바탕으로 2025년 개발자가 주목해야 할 기술 트렌드를 정리해줘.
%s
형식:
1. 핵심 트렌드 TOP 5 (간결하게)
2. 당장 배워야 할 기술
3. 주목할 오픈소스/도구
4. 한국 개발 시장 시사점`,
			},
			// ── 마케터 (20개) ───────────────────────────────────────
			"mkt_trend_analysis": {
				title: "트렌드 분석",
				searches: []struct{ name, q string }{
					{"이번 주 소비자 트렌드", "소비자 트렌드 2025 " + query},
					{"SNS 트렌드 분석", "TikTok Instagram 트렌드 viral 2025"},
					{"시장 분석", query + " 시장 트렌드 인사이트 2025"},
				},
				prompt: `다음 정보를 바탕으로 트렌드 분석 인사이트 리포트를 작성해줘.
%s
형식:
1. 이번 주 핵심 트렌드 TOP 5
2. SNS 플랫폼별 트렌드 키워드 (TikTok/Instagram/YouTube)
3. 소비자 행동 변화 인사이트
4. 마케터가 지금 당장 활용할 수 있는 액션 3가지
5. 다음 주 주목해야 할 트렌드 예측`,
			},
			"mkt_content_idea": {
				title: "콘텐츠 아이디어 브레인스토밍",
				searches: []struct{ name, q string }{
					{"SNS 인기 콘텐츠 유형", query + " SNS 콘텐츠 트렌드 2025"},
					{"바이럴 콘텐츠 사례", "바이럴 마케팅 성공 사례 2025 인스타 유튜브"},
				},
				prompt: `다음 정보를 바탕으로 콘텐츠 아이디어 10개를 브레인스토밍해줘.
%s
형식:
1. 인스타그램 릴스 아이디어 3개 (오프닝 훅 문구 포함)
2. 유튜브/숏폼 아이디어 3개 (제목 + 첫 3초 스크립트)
3. 블로그/뉴스레터 아이디어 2개
4. TikTok 트렌드 아이디어 2개
5. 각 아이디어별 예상 반응 포인트`,
			},
			"mkt_competitor_monitor": {
				title: "경쟁사 모니터링",
				searches: []struct{ name, q string }{
					{"경쟁사 최신 뉴스", query + " 경쟁사 마케팅 캠페인 뉴스 2025"},
					{"경쟁사 SNS 전략", query + " 경쟁사 SNS 활동 콘텐츠"},
					{"시장 점유율", query + " 시장 점유율 경쟁 현황"},
				},
				prompt: `다음 정보를 바탕으로 경쟁사 모니터링 주간 리포트를 작성해줘.
%s
형식:
1. 경쟁사 주요 활동 요약 (이번 주)
2. SNS 채널별 성과 비교
3. 신규 캠페인/프로모션 분석
4. 우리 브랜드 대비 강점/약점
5. 즉시 대응해야 할 액션 아이템`,
			},
			"mkt_ad_copy": {
				title: "광고 문구 생성",
				searches: []struct{ name, q string }{
					{"광고 카피라이팅 기법", "advertising copywriting hook formula 2025"},
					{"고성과 광고 문구 사례", query + " 광고 카피 성공 사례"},
				},
				prompt: `다음 정보를 바탕으로 A/B 테스트용 광고 문구 5개 버전을 생성해줘.
%s
형식:
[버전 A] 감성 소구형
- 헤드라인:
- 서브카피:
- CTA:

[버전 B] 혜택 중심형
[버전 C] 긴급성/희소성 자극형
[버전 D] 사회적 증거형
[버전 E] 질문형 훅

각 버전별 타겟 심리 설명 포함`,
			},
			"mkt_sns_post": {
				title: "SNS 게시물 전체 생성",
				searches: []struct{ name, q string }{
					{"SNS 게시물 최적 형식", "social media post best practice engagement 2025"},
					{"해시태그 전략", query + " Instagram TikTok hashtag strategy"},
				},
				prompt: `다음 정보를 바탕으로 SNS 게시물 전체를 생성해줘.
%s
형식:
📱 인스타그램
- 메인 문구 (150자 이내):
- 캡션 (500자 이내):
- 해시태그 20개:
- 게시 최적 시간:

🎵 TikTok/릴스
- 오프닝 훅 (3초):
- 스크립트 (30초):
- 트렌드 사운드 추천:

💼 LinkedIn
- 전문가 톤 게시물:`,
			},
			"mkt_campaign_plan": {
				title: "마케팅 캠페인 기획",
				searches: []struct{ name, q string }{
					{"마케팅 캠페인 기획 방법", "marketing campaign planning framework 2025"},
					{"성공적인 캠페인 사례", query + " marketing campaign success case study"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 캠페인 기획서를 작성해줘.
%s
형식:
1. 캠페인 목표 및 KPI 설정
2. 타겟 오디언스 정의 (페르소나)
3. 핵심 메시지 및 USP
4. 채널별 전략 (SNS/검색광고/이메일/오프라인)
5. 콘텐츠 캘린더 (4주 플랜)
6. 예산 배분 계획
7. 성과 측정 방법`,
			},
			"mkt_performance_report": {
				title: "마케팅 성과 리포트",
				searches: []struct{ name, q string }{
					{"마케팅 KPI 벤치마크", "marketing KPI benchmark 2025 CTR CVR ROAS"},
					{"디지털 마케팅 성과 분석", "digital marketing performance analysis report"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 성과 리포트 템플릿을 작성해줘.
%s
형식:
1. 이번 달 핵심 지표 요약
   - CTR / CVR / ROAS / CPA / CAC
2. 채널별 성과 비교 (Meta/Google/TikTok/이메일)
3. 업계 벤치마크 대비 성과
4. 잘된 캠페인 TOP 3 분석
5. 개선이 필요한 영역
6. 다음 달 액션 플랜`,
			},
			"mkt_seo_keyword": {
				title: "SEO 키워드 분석",
				searches: []struct{ name, q string }{
					{"SEO 키워드 트렌드", query + " SEO keyword search volume 2025"},
					{"롱테일 키워드", query + " long tail keyword low competition"},
					{"경쟁 키워드 분석", query + " competitor SEO keyword ranking"},
				},
				prompt: `다음 정보를 바탕으로 SEO 키워드 분석 리포트를 작성해줘.
%s
형식:
1. 핵심 키워드 TOP 10 (검색량/경쟁도)
2. 즉시 공략 가능한 롱테일 키워드 10개
3. 경쟁사가 사용하는 키워드 분석
4. 콘텐츠 주제 추천 (키워드 기반)
5. SEO 최적화 체크리스트
6. 월별 키워드 전략 로드맵`,
			},
			"mkt_email_newsletter": {
				title: "뉴스레터 작성",
				searches: []struct{ name, q string }{
					{"뉴스레터 트렌드", "email newsletter best practice open rate 2025"},
					{"뉴스레터 주제", query + " 뉴스레터 콘텐츠 트렌드"},
				},
				prompt: `다음 정보를 바탕으로 뉴스레터 초안을 작성해줘.
%s
형식:
📧 제목 라인 3가지 (A/B 테스트용)
📧 프리헤더 텍스트

## 뉴스레터 본문
1. 오프닝 훅 (2-3문장)
2. 메인 콘텐츠 (핵심 가치 전달)
3. 큐레이션 섹션 (이번 주 추천 3가지)
4. CTA (행동 유도)
5. 클로징 문구

디자인 가이드:
- 추천 이미지 배치
- 색상/폰트 방향`,
			},
			"mkt_influencer_search": {
				title: "인플루언서 검색",
				searches: []struct{ name, q string }{
					{"인플루언서 마케팅 트렌드", "influencer marketing trend 2025 Korea"},
					{"인플루언서 찾는 방법", query + " influencer Instagram TikTok YouTube"},
				},
				prompt: `다음 정보를 바탕으로 인플루언서 검색 및 협업 가이드를 작성해줘.
%s
형식:
1. 타겟에 맞는 인플루언서 유형 정의
   - 나노(1천~1만) / 마이크로(1만~10만) / 매크로(10만+) 구분
2. 플랫폼별 탐색 방법 (Instagram/TikTok/YouTube)
3. 인플루언서 평가 기준 (참여율/팔로워 품질/콘텐츠 방향성)
4. 협업 제안 DM/이메일 템플릿
5. 예산별 협업 전략
6. 계약 시 주의사항`,
			},
			"mkt_ab_test_idea": {
				title: "A/B 테스트 아이디어",
				searches: []struct{ name, q string }{
					{"A/B 테스트 방법론", "A/B testing best practices marketing 2025"},
					{"전환율 최적화", "conversion rate optimization CRO tips"},
				},
				prompt: `다음 정보를 바탕으로 A/B 테스트 아이디어 3세트를 제안해줘.
%s
형식:
[테스트 세트 1] 광고 소재
- A안:
- B안:
- 측정 지표:
- 예상 기간:

[테스트 세트 2] 랜딩페이지
- 테스트 요소: (헤드라인/CTA/이미지/레이아웃)
- A안 / B안:
- 성공 기준:

[테스트 세트 3] 이메일 캠페인
- 테스트 요소: (제목/발송시간/CTA)
- A/B 구성:

A/B 테스트 진행 원칙 5가지 포함`,
			},
			"mkt_hashtag_generator": {
				title: "해시태그 생성",
				searches: []struct{ name, q string }{
					{"트렌딩 해시태그", query + " trending hashtag Instagram TikTok 2025"},
					{"해시태그 전략", "hashtag strategy Instagram reach engagement"},
				},
				prompt: `다음 정보를 바탕으로 최적 해시태그 20개를 생성해줘.
%s
형식:
🔥 고볼륨 해시태그 (5개, 100만+ 게시물):
📈 중볼륨 해시태그 (8개, 10만~100만):
🎯 저볼륨 틈새 해시태그 (7개, 1만~10만):

플랫폼별 추천:
- Instagram: 상위 15개
- TikTok: 상위 10개
- LinkedIn: 상위 5개

해시태그 사용 팁 3가지 포함`,
			},
			"mkt_landing_page_copy": {
				title: "랜딩페이지 문구",
				searches: []struct{ name, q string }{
					{"랜딩페이지 카피라이팅", "landing page copywriting conversion 2025"},
					{"고전환율 랜딩페이지", query + " landing page high conversion example"},
				},
				prompt: `다음 정보를 바탕으로 랜딩페이지 문구를 작성해줘.
%s
형식:
🎯 히어로 섹션
- 헤드라인 (3가지 버전):
- 서브헤드라인:
- CTA 버튼 문구 (3가지):

💡 혜택 섹션 (3가지 핵심 혜택)
- 아이콘 + 제목 + 설명

🌟 소셜 프루프 섹션
- 추천사 문구 스타일

❓ FAQ 섹션 (5개)

📞 최종 CTA 섹션
- 긴급성/희소성 문구:`,
			},
			"mkt_social_calendar": {
				title: "소셜 미디어 캘린더",
				searches: []struct{ name, q string }{
					{"SNS 게시 최적 시간", "social media posting optimal time 2025"},
					{"콘텐츠 캘린더 템플릿", "social media content calendar template"},
				},
				prompt: `다음 정보를 바탕으로 1주일 소셜 미디어 게시 계획표를 작성해줘.
%s
형식:
| 날짜 | 플랫폼 | 콘텐츠 유형 | 주제/키워드 | 게시 시간 | 담당자 |
|------|--------|------------|------------|----------|--------|
월요일~일요일 7일 계획

추가:
- 플랫폼별 최적 게시 시간
- 이번 주 활용할 트렌딩 사운드/해시태그
- 예약 게시 도구 추천 (Buffer/Hootsuite/Meta Business)`,
			},
			"mkt_budget_plan": {
				title: "마케팅 예산 계획",
				searches: []struct{ name, q string }{
					{"디지털 광고 단가 2025", "digital advertising CPM CPC benchmark 2025 Korea"},
					{"마케팅 예산 배분 전략", "marketing budget allocation strategy ROI"},
				},
				prompt: `다음 정보를 바탕으로 마케팅 예산 계획을 작성해줘.
%s
형식:
1. 목표 기반 예산 산정 방법
2. 채널별 예산 배분 추천 (%)
   - 검색광고 / SNS광고 / 콘텐츠 / 인플루언서 / 오프라인
3. 채널별 예상 성과 (CPM/CPC/CPA 기준)
4. 월별 예산 집행 계획
5. ROI 측정 방법
6. 예산 절감 팁 3가지`,
			},
			"mkt_viral_content": {
				title: "바이럴 콘텐츠 전략",
				searches: []struct{ name, q string }{
					{"바이럴 콘텐츠 공식", "viral content formula psychology 2025"},
					{"바이럴 성공 사례", query + " viral marketing campaign success 2025"},
				},
				prompt: `다음 정보를 바탕으로 바이럴 가능성 높은 콘텐츠 전략을 작성해줘.
%s
형식:
1. 바이럴 공식 분석 (감정 자극 유형별)
   - 분노/감동/놀라움/웃음/공감
2. 지금 당장 실행 가능한 바이럴 포맷 3가지
3. 콘텐츠 훅 문구 5개 (복사 가능)
4. 공유를 유도하는 심리적 트리거
5. 플랫폼별 바이럴 최적화 방법
6. 바이럴 후 팔로업 전략`,
			},
			"mkt_customer_insight": {
				title: "고객 인사이트 분석",
				searches: []struct{ name, q string }{
					{"소비자 트렌드 분석", query + " 소비자 인사이트 행동 패턴 2025"},
					{"고객 리뷰 분석", query + " 고객 리뷰 불만 만족 분석"},
				},
				prompt: `다음 정보를 바탕으로 고객 인사이트 분석 리포트를 작성해줘.
%s
형식:
1. 핵심 타겟 페르소나 정의 (3가지)
2. 고객 Pain Point TOP 5
3. 구매 결정 요인 분석
4. 고객 여정 맵 (인지→고려→구매→재구매)
5. 리뷰/VOC에서 발견한 인사이트
6. 마케팅 메시지 방향 제안`,
			},
			"mkt_brand_voice": {
				title: "브랜드 보이스 유지",
				searches: []struct{ name, q string }{
					{"브랜드 보이스 가이드", "brand voice tone of voice guide example"},
					{"브랜드 일관성 전략", "brand consistency social media content strategy"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 보이스 가이드를 작성해줘.
%s
형식:
1. 브랜드 보이스 핵심 키워드 5개
2. 톤 스펙트럼 정의
   - 공식적 ←→ 친근한 / 진지한 ←→ 유머러스
3. 상황별 커뮤니케이션 톤 가이드
   - 일반 게시물 / 고객 응대 / 위기 상황 / 프로모션
4. 사용해야 할 표현 vs 피해야 할 표현
5. 채널별 톤 차이 (Instagram vs LinkedIn vs TikTok)
6. 브랜드 보이스 체크리스트`,
			},
			"mkt_weekly_digest": {
				title: "주간 마케팅 요약",
				searches: []struct{ name, q string }{
					{"마케팅 주간 트렌드", "digital marketing weekly digest trends 2025"},
					{"SNS 알고리즘 업데이트", "social media algorithm update 2025"},
				},
				prompt: `다음 정보를 바탕으로 이번 주 마케팅 한 장 요약을 작성해줘.
%s
형식:
📊 이번 주 마케팅 핵심 요약

✅ 완료한 캠페인/활동
📈 주요 성과 지표
🔥 이번 주 트렌드 & 알고리즘 변화
⚠️ 이슈 및 개선 필요 사항
📅 다음 주 예정 활동
💡 팀 공유 인사이트`,
			},
			"mkt_personal_brand": {
				title: "개인 브랜딩 콘텐츠",
				searches: []struct{ name, q string }{
					{"개인 브랜딩 전략", "personal branding LinkedIn content strategy 2025"},
					{"마케터 개인 브랜드", "marketer personal brand thought leadership"},
				},
				prompt: `다음 정보를 바탕으로 개인 브랜딩 콘텐츠를 작성해줘.
%s
형식:
💼 LinkedIn 게시물
- 전문성을 드러내는 인사이트 포스트 (300자):
- 경험 스토리 포스트 (500자):

📝 블로그/브런치 아티클
- 제목 3개 제안:
- 서론 초안 (200자):

🧵 스레드/X 스레드
- 10개 트윗 구성:

📌 개인 브랜딩 전략 팁
- 차별화 포인트 정의:
- 콘텐츠 주기 추천:`,
			},
			// ── 영업 (20개) ─────────────────────────────────────
			"sales_email_draft": {
				title: "영업 이메일 초안",
				searches: []struct{ name, q string }{
					{"영업 이메일 베스트 프랙티스", "sales email best practice cold outreach 2025"},
					{"B2B 이메일 템플릿", "B2B sales email template high response rate"},
				},
				prompt: `다음 정보를 바탕으로 고객 맞춤형 영업 이메일 초안을 작성해줘.
%s
형식:
제목 라인 3가지 (A/B/C):

[메인 초안]
안녕하세요, [고객명] 님.

1. 오프닝 (공감/칭찬/공통점)
2. 핵심 가치 제안 (2-3문장)
3. 사회적 증거
4. CTA (다음 단계 제안)
5. 클로징

[후속 버전] (3일 후 발송용)`,
			},
			"sales_meeting_prep": {
				title: "미팅 준비",
				searches: []struct{ name, q string }{
					{"고객사 정보", query + " 회사 정보 뉴스 2025"},
					{"영업 미팅 전략", "B2B 영업 미팅 성공 전략 준비"},
				},
				prompt: `다음 정보를 바탕으로 영업 미팅 준비 브리핑을 작성해줘.
%s
형식:
1. 고객사 현황 요약 (업계/규모/최근 뉴스)
2. 예상 Pain Point 3가지
3. 준비할 질문 목록 5가지
4. 미팅 오프닝 스크립트 (30초)
5. 예상 이의제기 & 대응
6. 다음 단계 클로징 멘트`,
			},
			"sales_followup": {
				title: "후속 메일 자동화",
				searches: []struct{ name, q string }{
					{"영업 후속 이메일", "sales followup email template after meeting"},
					{"후속 연락 타이밍", "sales followup timing best practice"},
				},
				prompt: `다음 정보를 바탕으로 미팅 후 후속 이메일과 일정 제안을 작성해줘.
%s
형식:
[당일 후속 메일]
- 제목:
- 본문: 감사 + 미팅 요약 + 다음 단계

[3일 후 메일]
- 제목:
- 본문: 가치 상기 + 자료 첨부 + CTA

[1주일 후 메일]
- 제목:
- 본문: 부드러운 압박 + 결정 지원

각 메일 최대 150자 이내`,
			},
			"sales_proposal": {
				title: "제안서 초안",
				searches: []struct{ name, q string }{
					{"성공적인 제안서 구조", "B2B proposal structure best practice 2025"},
					{"고객 니즈 분석", query + " 고객 pain point 솔루션"},
				},
				prompt: `다음 정보를 바탕으로 영업 제안서 초안을 작성해줘.
%s
형식:
1. Executive Summary (1페이지 요약)
2. 고객 현황 및 문제 정의
3. 우리의 솔루션 (핵심 가치 3가지)
4. 기대 효과 (정량적 수치 포함)
5. 도입 프로세스 (단계별 타임라인)
6. 가격/조건 제안 프레임
7. 다음 단계 (CTA)`,
			},
			"sales_objection": {
				title: "이의제기 대응 스크립트",
				searches: []struct{ name, q string }{
					{"영업 이의 대응", "sales objection handling script 2025"},
					{"가격 협상 전략", "price negotiation sales psychology"},
				},
				prompt: `다음 정보를 바탕으로 이의제기 대응 스크립트 5개를 작성해줘.
%s
형식:
1. "비싸요" → 공감 + 가치 재정의 + 대안 제시
2. "지금은 아닌 것 같아요" → 타이밍 이슈 대응
3. "경쟁사가 더 좋아요" → 차별화 포인트 강조
4. "내부 검토가 필요해요" → 의사결정 가속화
5. "기능이 부족해요" → 로드맵 제시 + 현재 가치
각 상황별 클로징 멘트 포함`,
			},
			"sales_pipeline": {
				title: "영업 파이프라인 정리",
				searches: []struct{ name, q string }{
					{"영업 파이프라인 관리", "sales pipeline management CRM best practice"},
					{"파이프라인 예측 방법", "sales pipeline forecast weighted probability"},
				},
				prompt: `다음 정보를 바탕으로 영업 파이프라인 정리 가이드를 작성해줘.
%s
형식:
1. 파이프라인 단계 정의 (리드→자격→제안→협상→클로즈)
2. 단계별 전환율 벤치마크
3. 정체 딜 식별 기준 및 액션
4. 이번 달 예상 매출 계산 방법
5. 파이프라인 건강도 체크리스트
6. CRM 업데이트 루틴 (일별/주별)`,
			},
			"sales_contract": {
				title: "계약서 초안",
				searches: []struct{ name, q string }{
					{"B2B 계약서 필수 항목", "B2B contract essential clauses 2025"},
					{"계약서 법적 체크", "sales contract legal review checklist Korea"},
				},
				prompt: `다음 정보를 바탕으로 영업 계약서 초안 구조를 작성해줘.
%s
형식:
1. 계약 당사자 정보
2. 서비스/제품 범위 (Scope of Work)
3. 납기 및 마일스톤
4. 대금 조건 (계약금/중도금/잔금)
5. 지적재산권 조항
6. 기밀유지 (NDA) 조항
7. 계약 해지 조건
8. 분쟁 해결 방법
※ 법률 검토 필수 안내 포함`,
			},
			"sales_discovery_question": {
				title: "고객 발견 질문 생성",
				searches: []struct{ name, q string }{
					{"영업 발견 질문", "sales discovery question SPIN selling 2025"},
					{"고객 니즈 파악 방법", "customer needs analysis question framework"},
				},
				prompt: `다음 정보를 바탕으로 고객 발견 질문 리스트를 작성해줘.
%s
형식:
[상황 질문 (Situation)] 5개
- 현재 상황 파악

[문제 질문 (Problem)] 5개
- 불편/고통 탐색

[시사 질문 (Implication)] 5개
- 문제의 파급 효과

[필요 질문 (Need-payoff)] 5개
- 해결 가치 확인

미팅 시작 아이스브레이킹 질문 3개 포함`,
			},
			"sales_demo_script": {
				title: "데모 스크립트 작성",
				searches: []struct{ name, q string }{
					{"제품 데모 스크립트", "product demo script best practice 2025"},
					{"데모 스토리텔링", "sales demo storytelling customer success"},
				},
				prompt: `다음 정보를 바탕으로 고객 맞춤 데모 대본을 작성해줘.
%s
형식:
[오프닝] (2분)
- 어젠다 설명
- 고객 상황 확인

[데모 본론] (15분)
- 핵심 기능 1: (스토리 + 시연)
- 핵심 기능 2:
- 핵심 기능 3:

[Q&A 대응 준비]
- 예상 질문 5개 + 답변

[클로징] (3분)
- 다음 단계 제안`,
			},
			"sales_negotiation": {
				title: "협상 전략 수립",
				searches: []struct{ name, q string }{
					{"영업 협상 전략", "sales negotiation strategy BATNA 2025"},
					{"가격 협상 심리", "price negotiation psychology anchoring"},
				},
				prompt: `다음 정보를 바탕으로 협상 전략과 시나리오를 작성해줘.
%s
형식:
1. 협상 전 준비 (목표/BATNA/양보 한계선)
2. 앵커링 전략 (첫 제안 설정)
3. 시나리오별 대응
   - 고객이 30% 할인 요구 시
   - 경쟁사 가격을 언급할 시
   - 결정권자가 없다고 할 시
4. 가치 교환 전술 (가격 대신 조건 협상)
5. 클로징 타이밍 포착 방법`,
			},
			"sales_forecast": {
				title: "영업 예측",
				searches: []struct{ name, q string }{
					{"영업 예측 방법", "sales forecasting method accuracy 2025"},
					{"파이프라인 예측 모델", "weighted pipeline forecast model"},
				},
				prompt: `다음 정보를 바탕으로 이번 달 영업 예측 리포트를 작성해줘.
%s
형식:
1. 예측 방법론 (가중 파이프라인/기대값)
2. 딜별 예상 매출 × 확률 계산
3. 낙관/현실/보수 시나리오
4. 목표 달성을 위한 갭 분석
5. 이번 달 반드시 클로즈할 딜 TOP 3
6. 다음 달 파이프라인 건강도 예측`,
			},
			"sales_crm_update": {
				title: "CRM 자동 업데이트",
				searches: []struct{ name, q string }{
					{"CRM 업데이트 베스트 프랙티스", "CRM data hygiene update best practice sales"},
				},
				prompt: `다음 정보를 바탕으로 CRM 업데이트 가이드와 템플릿을 작성해줘.
%s
형식:
1. 미팅 후 즉시 입력할 필드 목록
2. 미팅 노트 표준 포맷
   - 참석자 / 핵심 논의 / 액션 아이템 / 다음 단계
3. 딜 상태 업데이트 기준
4. CRM 데이터 정확도 유지 루틴 (주 1회)
5. 파이프라인 자동화 활용 팁`,
			},
			"sales_call_summary": {
				title: "영업 통화 요약",
				searches: []struct{ name, q string }{
					{"영업 통화 요약 방법", "sales call summary template action items"},
				},
				prompt: `다음 정보를 바탕으로 영업 통화 요약 템플릿을 작성해줘.
%s
형식:
📞 통화 요약
- 일시 / 참석자:
- 통화 목적:

💬 핵심 논의 내용 (불릿 3-5개)

⚡ 고객이 표현한 Pain Point

✅ 합의된 사항

📋 액션 아이템
| 항목 | 담당 | 기한 |

📅 다음 단계`,
			},
			"sales_proposal_followup": {
				title: "제안서 후속 관리",
				searches: []struct{ name, q string }{
					{"제안서 후속 전략", "proposal followup strategy win rate"},
					{"제안 후 의사결정 지원", "after proposal decision making support"},
				},
				prompt: `다음 정보를 바탕으로 제안서 발송 후 후속 관리 플랜을 작성해줘.
%s
형식:
[D+1] 확인 연락
- 수신 확인 + 질문 유도

[D+3] 가치 보강
- 추가 자료 / 케이스 스터디 전달

[D+7] 의사결정 지원
- 내부 검토 지원 자료 제공

[D+14] 부드러운 압박
- 타이밍 이슈 대응

각 단계별 이메일/문자 초안 포함`,
			},
			"sales_win_loss_analysis": {
				title: "Win/Loss 분석",
				searches: []struct{ name, q string }{
					{"Win Loss 분석 방법", "win loss analysis sales learning 2025"},
					{"영업 패배 원인 분석", "sales lost deal analysis reason"},
				},
				prompt: `다음 정보를 바탕으로 Win/Loss 분석 리포트를 작성해줘.
%s
형식:
1. 분석 기간 및 대상 딜 개요
2. Win 패턴 분석
   - 공통 승리 요인 TOP 5
   - 자주 이긴 산업/고객 유형
3. Loss 패턴 분석
   - 주요 패인 원인 TOP 5
   - 경쟁사에게 진 이유
4. 개선 액션 플랜 3가지
5. 다음 분기 전략 방향`,
			},
			"sales_referral_request": {
				title: "추천 요청 메시지",
				searches: []struct{ name, q string }{
					{"고객 추천 요청", "customer referral request script best practice"},
					{"레퍼럴 마케팅 전략", "referral marketing B2B strategy"},
				},
				prompt: `다음 정보를 바탕으로 고객 추천 요청 메시지를 작성해줘.
%s
형식:
[이메일 버전]
제목:
본문: 관계 상기 → 만족도 확인 → 추천 요청 → 인센티브 → CTA

[문자/카카오 버전] (80자 이내)

[전화 스크립트] (30초)

추천 요청 최적 타이밍 가이드 포함`,
			},
			"sales_price_negotiation": {
				title: "가격 협상 전략",
				searches: []struct{ name, q string }{
					{"가격 협상 전략", "pricing negotiation strategy 2025"},
					{"할인 정책 가이드", "discount policy sales negotiation framework"},
				},
				prompt: `다음 정보를 바탕으로 가격 협상 전략을 작성해줘.
%s
형식:
1. 가격 방어 프레임워크 (가치 기반 대응)
2. 할인 제공 시 조건 교환 전술
   - "할인 대신 조건을 바꾸는 법"
3. 가격 앵커링 설정 방법
4. 번들링/패키지 재구성 전략
5. 협상 불가 선언 타이밍
6. 최종 제안 클로징 멘트`,
			},
			"sales_contract_review": {
				title: "계약서 검토",
				searches: []struct{ name, q string }{
					{"계약서 위험 항목", "contract red flags review checklist B2B"},
					{"계약서 협상 포인트", "contract negotiation points sales"},
				},
				prompt: `다음 정보를 바탕으로 계약서 검토 체크리스트를 작성해줘.
%s
형식:
🔴 즉시 수정 필요 (위험 항목)
- 과도한 면책 조항
- 일방적 해지 권리
- 무제한 손해배상

🟡 협상 권장 항목
- 납기 지연 패널티 기준
- 범위 변경(Change Order) 절차
- 지식재산권 귀속

🟢 확인 필수 항목
- 준거법 및 관할법원
- 갱신/연장 조건

협상 가이드 포함, ※ 법률 검토 필수 안내`,
			},
			"sales_quarterly_review": {
				title: "분기 영업 리뷰",
				searches: []struct{ name, q string }{
					{"분기 영업 리뷰 방법", "quarterly sales review QBR template"},
					{"영업 성과 분석", "sales performance analysis quarterly"},
				},
				prompt: `다음 정보를 바탕으로 분기 영업 리뷰 리포트를 작성해줘.
%s
형식:
1. 분기 실적 요약 (목표 vs 실제)
2. 채널/제품별 성과 분석
3. Win/Loss 비율 및 원인
4. 파이프라인 건강도
5. 팀별/개인별 성과 하이라이트
6. 다음 분기 전략 및 목표 설정
7. 경영진 보고용 요약 (1페이지)`,
			},
			"sales_client_portrait": {
				title: "고객 프로필 분석",
				searches: []struct{ name, q string }{
					{"고객 프로파일링 방법", "customer profiling ICP ideal customer profile"},
					{"B2B 구매자 페르소나", "B2B buyer persona decision maker analysis"},
				},
				prompt: `다음 정보를 바탕으로 고객 프로필 분석 리포트를 작성해줘.
%s
형식:
1. 기업 프로파일 (규모/업종/성장단계)
2. 의사결정 구조 (Champion/Blocker/Budget)
3. 핵심 Pain Point 및 우선순위
4. 구매 프로세스 및 타임라인
5. 경쟁사 대비 포지셔닝
6. 맞춤형 접근 전략 3가지`,
			},
			// ── PM (20개) ────────────────────────────────────────
			"pm_requirements": {
				title: "요구사항 정리",
				searches: []struct{ name, q string }{
					{"요구사항 정리 방법", "product requirements gathering template PRD"},
					{"기능 명세서 작성", "functional specification document template"},
				},
				prompt: `다음 정보를 바탕으로 요구사항 정리 문서를 작성해줘.
%s
형식:
1. 배경 및 목표 (Why)
2. 요구사항 수집 방법 (인터뷰/설문/분석)
3. 기능 요구사항 목록 (Must/Should/Could/Won't)
4. 비기능 요구사항 (성능/보안/UX)
5. 우선순위 결정 기준
6. 이해관계자 승인 절차`,
			},
			"pm_roadmap": {
				title: "로드맵 업데이트",
				searches: []struct{ name, q string }{
					{"제품 로드맵 작성", "product roadmap template best practice 2025"},
					{"로드맵 우선순위", "roadmap prioritization framework OKR"},
				},
				prompt: `다음 정보를 바탕으로 제품 로드맵을 작성해줘.
%s
형식:
| 분기 | 테마 | 기능 | 목표 지표 | 상태 |

Now (이번 분기):
Next (다음 분기):
Later (그 이후):

우선순위 결정 근거:
- 비즈니스 임팩트
- 개발 복잡도
- 사용자 요청 빈도

이해관계자 커뮤니케이션 가이드 포함`,
			},
			"pm_stakeholder_summary": {
				title: "이해관계자 브리핑",
				searches: []struct{ name, q string }{
					{"이해관계자 브리핑 방법", "stakeholder briefing communication executive summary"},
				},
				prompt: `다음 정보를 바탕으로 이해관계자 브리핑 문서를 작성해줘.
%s
형식:
📋 이번 주 요약 (Executive 1-pager)

✅ 완료된 주요 결정사항
⚡ 현재 진행 중인 이슈
⚠️ 리스크 및 블로커
📊 핵심 지표 현황
📅 다음 주 주요 마일스톤
❓ 이해관계자 결정 필요 사항

커뮤니케이션 채널별 요약 길이 가이드`,
			},
			"pm_risk_analysis": {
				title: "리스크 분석",
				searches: []struct{ name, q string }{
					{"프로젝트 리스크 분석", "project risk analysis framework RAID log"},
					{"리스크 대응 전략", "risk mitigation strategy product management"},
				},
				prompt: `다음 정보를 바탕으로 리스크 분석 리포트를 작성해줘.
%s
형식:
| 리스크 | 발생확률 | 영향도 | 위험도 | 대응 방안 | 담당자 |

카테고리별 분류:
1. 기술 리스크
2. 일정 리스크
3. 자원 리스크
4. 외부/시장 리스크

조기 경보 지표 (Early Warning Signals) 설정
리스크 모니터링 주기 및 방법`,
			},
			"pm_meeting_note": {
				title: "미팅 노트 정리",
				searches: []struct{ name, q string }{
					{"미팅 노트 작성 방법", "meeting notes template action items best practice"},
				},
				prompt: `다음 정보를 바탕으로 미팅 노트를 정리해줘.
%s
형식:
📅 미팅 정보
- 일시 / 참석자 / 목적

💬 논의 내용 (주제별)

✅ 결정 사항

📋 액션 아이템
| 항목 | 담당자 | 기한 | 상태 |

❓ 미해결 질문 / 다음 미팅 어젠다

배포 대상: [참석자/이해관계자]`,
			},
			"pm_user_story": {
				title: "유저 스토리 작성",
				searches: []struct{ name, q string }{
					{"유저 스토리 작성법", "user story writing acceptance criteria example"},
					{"Agile 스토리 포인트", "agile story point estimation planning poker"},
				},
				prompt: `다음 정보를 바탕으로 유저 스토리와 Acceptance Criteria를 작성해줘.
%s
형식:
[유저 스토리]
As a [사용자 유형]
I want to [기능/행동]
So that [얻는 가치]

[Acceptance Criteria]
Given [전제 조건]
When [행동]
Then [결과]

스토리 포인트 추정: [1/2/3/5/8]
우선순위: [Must/Should/Could]
의존성: [관련 스토리]

3-5개 스토리 생성`,
			},
			"pm_weekly_report": {
				title: "주간 보고서",
				searches: []struct{ name, q string }{
					{"PM 주간 보고서", "product manager weekly report template"},
					{"스프린트 리뷰", "sprint review retrospective template"},
				},
				prompt: `다음 정보를 바탕으로 PM 주간 보고서를 작성해줘.
%s
형식:
1. 이번 주 완료 항목 (Done)
2. 진행 중 항목 및 이슈 (In Progress)
3. 다음 주 계획 (Todo)
4. 리스크 및 블로커
5. 주요 지표 현황
6. 이해관계자 전달 사항`,
			},
			"pm_prd_write": {
				title: "PRD 작성",
				searches: []struct{ name, q string }{
					{"PRD 작성 방법", "PRD product requirements document template 2025"},
					{"사용자 스토리 PRD", "user story acceptance criteria PRD example"},
				},
				prompt: `다음 정보를 바탕으로 PRD(제품 요구사항 문서) 초안을 작성해줘.
%s
형식:
# PRD: [제품/기능명]

## 1. 배경 및 목표 (WHY)
## 2. 대상 사용자 및 페르소나
## 3. 핵심 기능 목록
| 기능 | 우선순위 | 설명 |
## 4. 비기능 요구사항
## 5. 성공 지표 (KPI)
## 6. 제외 범위 (Out of Scope)
## 7. 의존성 및 위험요소
## 8. 타임라인`,
			},
			"pm_spec_review": {
				title: "기획서 검토",
				searches: []struct{ name, q string }{
					{"기획서 검토 기준", "product spec review checklist feedback"},
					{"기획 완성도 평가", "product requirements completeness review"},
				},
				prompt: `다음 정보를 바탕으로 기획서 검토 피드백을 작성해줘.
%s
형식:
✅ 잘된 점

❌ 보완 필요 사항
- 명확하지 않은 요구사항
- 누락된 엣지 케이스
- 기술적 실현 가능성 이슈
- UX 흐름 개선 포인트

💡 개선 제안 (우선순위별)

❓ 추가 확인 필요 질문 (개발팀/디자인팀)

종합 평가: [완성도 점수/100]`,
			},
			"pm_priority_matrix": {
				title: "우선순위 매트릭스",
				searches: []struct{ name, q string }{
					{"우선순위 매트릭스", "priority matrix RICE MoSCoW framework"},
					{"제품 우선순위 결정", "product prioritization method impact effort"},
				},
				prompt: `다음 정보를 바탕으로 우선순위 매트릭스를 작성해줘.
%s
형식:
[MoSCoW 분류]
Must Have: (필수)
Should Have: (권장)
Could Have: (여유되면)
Won't Have: (이번 버전 제외)

[RICE 점수 계산]
| 기능 | Reach | Impact | Confidence | Effort | RICE |

[2×2 매트릭스]
- 높은 임팩트 + 낮은 노력 → 즉시 실행
- 높은 임팩트 + 높은 노력 → 계획 수립
- 낮은 임팩트 + 낮은 노력 → 틈새 실행
- 낮은 임팩트 + 높은 노력 → 제외`,
			},
			"pm_retrospective": {
				title: "회고 미팅 정리",
				searches: []struct{ name, q string }{
					{"회고 미팅 방법", "retrospective meeting template Start Stop Continue"},
					{"애자일 회고 기법", "agile retrospective techniques team"},
				},
				prompt: `다음 정보를 바탕으로 회고 미팅 정리 문서를 작성해줘.
%s
형식:
[Start - Stop - Continue]
✨ Start (새로 시작할 것):
🛑 Stop (그만할 것):
✅ Continue (계속할 것):

[주요 인사이트]

[액션 아이템]
| 항목 | 담당자 | 기한 |

[팀 건강도 체크]
- 협업: /10
- 커뮤니케이션: /10
- 기술적 품질: /10

다음 스프린트 개선 포인트 TOP 3`,
			},
			"pm_okr_setting": {
				title: "OKR 설정",
				searches: []struct{ name, q string }{
					{"OKR 작성 방법", "OKR objective key results writing best practice"},
					{"OKR 사례", "OKR examples product team 2025"},
				},
				prompt: `다음 정보를 바탕으로 OKR을 작성해줘.
%s
형식:
## Objective (목표)
[야심차고 영감을 주는 질적 목표]

## Key Results (핵심 결과)
KR1: [측정 가능한 수치 목표]
KR2: [측정 가능한 수치 목표]
KR3: [측정 가능한 수치 목표]

## 이니셔티브 (실행 과제)
KR별 핵심 액션 2-3개

OKR 작성 원칙:
- Objective: 동기부여 + 방향 제시
- KR: 숫자로 측정 가능
- 달성률 70%가 좋은 OKR

분기별 체크인 방법 포함`,
			},
			"pm_resource_plan": {
				title: "리소스 계획",
				searches: []struct{ name, q string }{
					{"리소스 계획 방법", "resource planning project management template"},
					{"인력 배치 최적화", "team capacity planning sprint allocation"},
				},
				prompt: `다음 정보를 바탕으로 리소스 계획을 작성해줘.
%s
형식:
1. 프로젝트 리소스 현황
| 역할 | 인원 | 가용 시간 | 현재 배치 |

2. 리소스 갭 분석 (부족/과잉)

3. 우선순위 기반 배치 방안

4. 외부 리소스 필요 여부 (외주/채용)

5. 리소스 충돌 해결 방법

6. 주별 용량 계획 (4주)`,
			},
			"pm_stakeholder_map": {
				title: "이해관계자 맵",
				searches: []struct{ name, q string }{
					{"이해관계자 분석", "stakeholder mapping analysis influence interest"},
					{"이해관계자 관리 전략", "stakeholder management strategy communication"},
				},
				prompt: `다음 정보를 바탕으로 이해관계자 맵을 작성해줘.
%s
형식:
[2×2 이해관계자 맵]
영향력 높음 + 관심 높음 → 긴밀 관리
영향력 높음 + 관심 낮음 → 만족 유지
영향력 낮음 + 관심 높음 → 정보 제공
영향력 낮음 + 관심 낮음 → 모니터링

| 이해관계자 | 역할 | 영향력 | 관심도 | 입장 | 관리 전략 |

커뮤니케이션 주기별 계획 포함`,
			},
			"pm_feature_kanban": {
				title: "기능 칸반 정리",
				searches: []struct{ name, q string }{
					{"칸반 보드 운영", "kanban board management workflow WIP limit"},
				},
				prompt: `다음 정보를 바탕으로 기능 칸반 정리 가이드를 작성해줘.
%s
형식:
[칸반 컬럼 구성]
📋 Backlog | 🔍 분석 중 | 🎨 디자인 | 💻 개발 | 🧪 QA | ✅ Done

[WIP 한계 설정]
각 컬럼별 최대 진행 항목 수

[백로그 → 칸반 분류 기준]
- 즉시 실행 가능한 카드 조건
- 카드 크기 기준 (1 스프린트 이내)
- 카드 작성 표준 포맷

[블로킹 카드 처리 방법]

주간 칸반 리뷰 루틴 포함`,
			},
			"pm_user_interview_summary": {
				title: "사용자 인터뷰 요약",
				searches: []struct{ name, q string }{
					{"사용자 인터뷰 분석", "user interview analysis affinity mapping insights"},
					{"인터뷰 인사이트 추출", "qualitative research synthesis themes"},
				},
				prompt: `다음 정보를 바탕으로 사용자 인터뷰 요약 리포트를 작성해줘.
%s
형식:
1. 인터뷰 개요 (대상/방법/일정)
2. 핵심 테마별 인사이트
3. 사용자 Pain Point TOP 5
4. 자주 언급된 키워드/표현
5. 예상과 달랐던 발견
6. 제품 개선 제안 (우선순위별)
7. 다음 인터뷰 질문 개선안`,
			},
			"pm_competitor_analysis": {
				title: "경쟁사 분석",
				searches: []struct{ name, q string }{
					{"경쟁사 제품 분석", query + " competitor product analysis 2025"},
					{"경쟁 포지셔닝", query + " competitive positioning feature comparison"},
				},
				prompt: `다음 정보를 바탕으로 경쟁 제품 분석 리포트를 작성해줘.
%s
형식:
[기능 비교표]
| 기능 | 우리 | 경쟁사 A | 경쟁사 B |

[포지셔닝 분석]
- 가격 포지셔닝
- 타겟 세그먼트
- 핵심 차별화 메시지

[우리의 강점/약점]

[기회 포착 포인트]

[전략적 방향 제안]`,
			},
			"pm_go_to_market": {
				title: "Go-to-Market 전략",
				searches: []struct{ name, q string }{
					{"GTM 전략 수립", "go-to-market strategy template product launch 2025"},
					{"제품 출시 전략", "product launch plan checklist B2B SaaS"},
				},
				prompt: `다음 정보를 바탕으로 Go-to-Market 전략을 작성해줘.
%s
형식:
1. 타겟 시장 정의 (TAM/SAM/SOM)
2. ICP (이상적 고객 프로필)
3. 가치 제안 (Value Proposition)
4. 가격 전략
5. 유통/채널 전략
6. 마케팅 & 영업 플레이북
7. 출시 타임라인 (T-4주 ~ 출시 후)
8. 성공 지표 (KPI)`,
			},
			"pm_sprint_planning": {
				title: "스프린트 계획",
				searches: []struct{ name, q string }{
					{"스프린트 계획 방법", "agile sprint planning best practice velocity"},
					{"스프린트 용량 산정", "sprint capacity planning story points"},
				},
				prompt: `다음 정보를 바탕으로 스프린트 계획 가이드를 작성해줘.
%s
형식:
1. 스프린트 목표 선언
2. 팀 용량 계산 (가용 시간 × 인원)
3. 백로그 선택 기준
4. 스토리 포인트 할당 방법
5. 스프린트 백로그 (확정 항목)
| 스토리 | 포인트 | 담당자 |
6. 데일리 스탠드업 루틴
7. 스프린트 리스크 체크`,
			},
			"pm_metrics_dashboard": {
				title: "지표 대시보드",
				searches: []struct{ name, q string }{
					{"제품 핵심 지표", "product metrics KPI dashboard 2025"},
					{"AARRR 지표", "AARRR pirate metrics product growth"},
				},
				prompt: `다음 정보를 바탕으로 PM 지표 대시보드 구성을 작성해줘.
%s
형식:
[핵심 지표 대시보드]
AARRR 퍼널:
- Acquisition: (신규 유입)
- Activation: (첫 경험 성공률)
- Retention: (재방문율)
- Revenue: (전환/결제)
- Referral: (추천)

[제품별 핵심 지표]
- DAU/MAU / NPS / CSAT
- 기능 채택률 / 완료율

[대시보드 시각화 추천]
[주간 지표 리뷰 루틴]`,
			},
			// ── 디자이너 (20개) ──────────────────────────────────
			"design_reference": {
				title: "레퍼런스 검색",
				searches: []struct{ name, q string }{
					{"디자인 레퍼런스", query + " design reference inspiration Dribbble Behance 2025"},
					{"UI 디자인 트렌드", "UI UX design trends 2025"},
				},
				prompt: `다음 정보를 바탕으로 디자인 레퍼런스 가이드를 작성해줘.
%s
형식:
1. 추천 레퍼런스 사이트/브랜드 TOP 5 (이유 포함)
2. Dribbble/Behance/Pinterest 검색 키워드
3. 현재 트렌드 키워드 5개
4. 색상 팔레트 방향 제안
5. 레퍼런스 수집 → 무드보드 구성 방법`,
			},
			"design_file_organize": {
				title: "디자인 파일 정리",
				searches: []struct{ name, q string }{
					{"디자인 파일 관리", "design file organization naming convention Figma"},
				},
				prompt: `다음 정보를 바탕으로 디자인 파일 정리 가이드를 작성해줘.
%s
형식:
1. 폴더 구조 설계
   /Projects/[클라이언트]/[프로젝트]/[버전]
2. 파일 네이밍 규칙
   YYYYMMDD_프로젝트명_버전_담당자
3. 에셋 분류 기준 (로고/아이콘/이미지/폰트)
4. 버전 관리 방법 (v1.0 / v1.1 / Final)
5. 아카이브 정책 (보관 기간/압축 방법)
6. 팀 공유 폴더 운영 방법`,
			},
			"design_color_palette": {
				title: "컬러 팔레트 생성",
				searches: []struct{ name, q string }{
					{"컬러 팔레트 이론", "color palette theory brand design 2025"},
					{"브랜드 컬러 선택", "brand color psychology selection guide"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 컬러 팔레트를 제안해줘.
%s
형식:
🎨 컬러 팔레트 제안

Primary Color:
- HEX: #______
- 심리적 의미:
- 사용 맥락:

Secondary Color: #______
Accent Color: #______

[명도 스케일] (Primary 기준)
100 / 200 / 300 / 400 / 500 / 600 / 700 / 800 / 900

[중립 컬러] (텍스트/배경용)
- 텍스트: #______
- 배경: #______
- 보더: #______

접근성 대비율 체크 (WCAG AA 기준)`,
			},
			"design_image_edit": {
				title: "이미지 일괄 편집",
				searches: []struct{ name, q string }{
					{"이미지 일괄 편집 방법", "batch image processing automation design workflow"},
				},
				prompt: `다음 정보를 바탕으로 이미지 일괄 편집 가이드를 작성해줘.
%s
형식:
1. 파일 형식 변환 규칙 (JPG/PNG/WebP/SVG)
2. 크기 규격 체계
   - 웹: 1920px / 1280px / 768px / 375px
   - SNS: 인스타(1080×1080) / 유튜브썸네일(1280×720)
3. 파일명 규칙 적용
4. 압축률 설정 (품질 vs 용량)
5. 메타데이터 처리 방법
6. 추천 툴 (ImageOptim/Squoosh/Sharp CLI)`,
			},
			"design_content_idea": {
				title: "콘텐츠 디자인 아이디어",
				searches: []struct{ name, q string }{
					{"포스터 디자인 트렌드", query + " poster design trend 2025"},
					{"콘텐츠 디자인 아이디어", "creative content design concept idea"},
				},
				prompt: `다음 정보를 바탕으로 콘텐츠 디자인 아이디어 5개 컨셉을 제안해줘.
%s
형식:
[컨셉 1] 타이틀
- 스타일: (미니멀/볼드/레트로 등)
- 컬러 방향:
- 레이아웃 구조:
- 필요 에셋:

[컨셉 2~5] 동일 형식

각 컨셉별 적합한 활용처 (웹/SNS/인쇄) 명시`,
			},
			"design_feedback": {
				title: "디자인 피드백",
				searches: []struct{ name, q string }{
					{"디자인 피드백 방법", "design critique feedback constructive method"},
					{"UI 디자인 평가 기준", "UI design evaluation heuristics Nielsen"},
				},
				prompt: `다음 정보를 바탕으로 구조화된 디자인 피드백을 작성해줘.
%s
형식:
✅ 잘된 점 (구체적으로)

🔴 개선 필요 사항
1. 시각적 계층구조 (Visual Hierarchy)
2. 색상 및 대비 (Color & Contrast)
3. 타이포그래피 일관성
4. 여백 및 정렬 (Spacing & Alignment)
5. 사용성 (Usability)

💡 구체적 개선 제안 (우선순위별)

📐 디자인 시스템 적용 여부 체크`,
			},
			"design_moodboard": {
				title: "무드보드 생성",
				searches: []struct{ name, q string }{
					{"무드보드 참고 이미지", query + " moodboard visual inspiration 2025"},
					{"컬러 분위기", query + " color mood aesthetic"},
				},
				prompt: `다음 정보를 바탕으로 무드보드 가이드를 작성해줘.
%s
형식:
🎨 무드보드 컨셉

핵심 키워드 (5개):

컬러 팔레트 방향:
- 메인 컬러: [분위기 설명 + 예시 HEX]
- 보조 컬러:
- 포인트 컬러:

타이포그래피 방향:
- 헤드라인 폰트 스타일:
- 본문 폰트 스타일:

이미지/텍스처 방향:
- 사진 분위기:
- 패턴/텍스처:

레퍼런스 수집 사이트 및 검색 키워드 5개`,
			},
			"design_ui_kit": {
				title: "UI Kit 가이드",
				searches: []struct{ name, q string }{
					{"UI Kit 구성 방법", "UI Kit component library design system 2025"},
					{"디자인 시스템 구조", "design system atomic design component"},
				},
				prompt: `다음 정보를 바탕으로 UI Kit 구성 가이드를 작성해줘.
%s
형식:
[기초 요소 (Foundations)]
- 컬러 시스템
- 타이포그래피 스케일
- 간격 시스템 (4px/8px 그리드)
- 아이콘 스타일

[컴포넌트 목록 (우선순위별)]
Tier 1 (필수): Button/Input/Card/Modal/Toast
Tier 2 (권장): Dropdown/Tabs/Badge/Avatar
Tier 3 (나중): DatePicker/Table/Chart

[Figma 구성 방법]
- 컴포넌트 → 인스턴스 구조
- 오토레이아웃 활용법
- 네이밍 규칙`,
			},
			"design_prototype_review": {
				title: "프로토타입 검토",
				searches: []struct{ name, q string }{
					{"프로토타입 검토 기준", "prototype review checklist usability UX"},
					{"UX 검토 방법", "UX heuristic evaluation prototype"},
				},
				prompt: `다음 정보를 바탕으로 프로토타입 검토 피드백을 작성해줘.
%s
형식:
[UX 흐름 검토]
✅ 자연스러운 플로우
❌ 끊기는 구간 및 원인

[닐슨 휴리스틱 10원칙 체크]
1. 시스템 상태 가시성
2. 사용자 제어 및 자유도
3. 일관성
(각 항목 통과/주의/실패 + 설명)

[모바일 친화성]
[접근성 기본 체크]

우선순위별 개선 제안 TOP 5`,
			},
			"design_asset_export": {
				title: "에셋 일괄 내보내기",
				searches: []struct{ name, q string }{
					{"디자인 에셋 내보내기", "design asset export specification Figma guide"},
				},
				prompt: `다음 정보를 바탕으로 에셋 내보내기 규칙을 작성해줘.
%s
형식:
[플랫폼별 내보내기 규격]
iOS:
- 1x / 2x / 3x (PNG)
- 아이콘: .pdf 또는 .svg

Android:
- mdpi / hdpi / xhdpi / xxhdpi / xxxhdpi
- 벡터: .xml (SVG 변환)

Web:
- SVG (아이콘/로고)
- WebP + JPG (사진)
- PNG (투명배경 필요시)

[파일명 규칙]
ic_이름_상태_크기.확장자

[Figma 내보내기 자동화 방법]`,
			},
			"design_brand_guideline": {
				title: "브랜드 가이드라인",
				searches: []struct{ name, q string }{
					{"브랜드 가이드라인 구성", "brand guideline template visual identity 2025"},
					{"브랜드 아이덴티티 사례", "brand identity guideline example"},
				},
				prompt: `다음 정보를 바탕으로 브랜드 가이드라인 구조를 작성해줘.
%s
형식:
1. 브랜드 스토리 & 철학
2. 로고 사용 규칙
   - 최소 크기 / 여백 / 금지 사례
3. 컬러 시스템 (Primary/Secondary/Neutral)
4. 타이포그래피 시스템
   - 헤드라인 / 서브 / 본문 / 캡션
5. 이미지 스타일 가이드
6. 아이콘 스타일
7. DO & DON'T 사례
8. 적용 예시 (명함/웹/SNS)`,
			},
			"design_social_media_kit": {
				title: "소셜 미디어 키트",
				searches: []struct{ name, q string }{
					{"SNS 디자인 규격", "social media design template size 2025"},
					{"소셜 키트 구성", "social media kit template brand"},
				},
				prompt: `다음 정보를 바탕으로 소셜 미디어 키트 구성 가이드를 작성해줘.
%s
형식:
[플랫폼별 규격]
Instagram: 정사각(1080×1080) / 세로(1080×1350) / 스토리(1080×1920)
YouTube: 썸네일(1280×720) / 채널아트(2560×1440)
LinkedIn: 포스트(1200×627)
TikTok: 세로(1080×1920)

[키트 구성 항목]
- 프로필 사진 프레임
- 포스트 템플릿 (일반/프로모션/인포그래픽)
- 스토리 템플릿
- 하이라이트 커버

[Canva/Figma 템플릿 구성 팁]`,
			},
			"design_presentation_deck": {
				title: "발표 자료 제작",
				searches: []struct{ name, q string }{
					{"발표 자료 디자인", "presentation deck design best practice storytelling"},
					{"슬라이드 구성 방법", "pitch deck slide structure compelling"},
				},
				prompt: `다음 정보를 바탕으로 발표 자료 구성 가이드를 작성해줘.
%s
형식:
[슬라이드 구성 (10-20장)]
1. 표지
2. 목차/어젠다
3. 문제 정의
4. 솔루션/핵심 메시지
5. 데이터/증거
6. 사례/케이스 스터디
7. 액션 플랜
8. Q&A / 마무리

[디자인 원칙]
- 슬라이드당 1개 메시지
- 텍스트 최소화 (키워드만)
- 데이터는 차트로 시각화

[발표자 노트 작성 팁]`,
			},
			"design_icon_set": {
				title: "아이콘 세트 가이드",
				searches: []struct{ name, q string }{
					{"아이콘 디자인 스타일", "icon design style guide 2025 outline filled"},
					{"무료 아이콘 리소스", "free icon set resource design 2025"},
				},
				prompt: `다음 정보를 바탕으로 아이콘 세트 제작 가이드를 작성해줘.
%s
형식:
[스타일 정의]
- Outline / Filled / Duo-tone 중 선택 이유
- 선 두께: ____px
- 코너 반경: ____px
- 그리드: 24px × 24px

[필수 아이콘 20개 목록]
카테고리별: 내비게이션/액션/상태/소셜

[일관성 체크리스트]
- 시각적 무게 균등
- 픽셀 맞춤 (Pixel Perfect)
- 의미 명확성

[리소스 추천]
Heroicons / Lucide / Phosphor Icons`,
			},
			"design_typography": {
				title: "타이포그래피 시스템",
				searches: []struct{ name, q string }{
					{"타이포그래피 시스템", "typography system scale design 2025"},
					{"한국어 폰트 추천", "Korean font recommendation web design 2025"},
				},
				prompt: `다음 정보를 바탕으로 타이포그래피 시스템을 작성해줘.
%s
형식:
[폰트 선택]
- 헤드라인: [폰트명] / 이유
- 본문: [폰트명] / 이유
- 모노스페이스: [폰트명] (코드용)
- 한국어: [폰트명]

[타입 스케일]
| 이름 | 크기 | 굵기 | 줄간격 | 용도 |
Display / H1 / H2 / H3 / Body-L / Body-M / Caption

[사용 규칙]
- 최대 폰트 종류: 2개
- 강조: Bold 사용 (이탤릭 최소화)
- 접근성: 최소 16px 본문

[웹폰트 로딩 최적화 방법]`,
			},
			"design_animation_idea": {
				title: "애니메이션 아이디어",
				searches: []struct{ name, q string }{
					{"UI 애니메이션 트렌드", "UI animation micro-interaction trend 2025"},
					{"Lottie 애니메이션 사례", "Lottie animation example UI motion design"},
				},
				prompt: `다음 정보를 바탕으로 UI 애니메이션 아이디어를 작성해줘.
%s
형식:
[마이크로 인터랙션 아이디어]
1. 버튼 클릭 피드백 (0.2s ease-out)
2. 로딩 상태 표현
3. 성공/실패 알림 애니메이션
4. 페이지 전환 효과
5. 스크롤 트리거 애니메이션

[Lottie 활용 포인트]
- 온보딩 캐릭터
- 빈 상태(Empty State) 일러스트
- 성공 축하 이펙트

[애니메이션 원칙]
- 지속시간: 200-500ms
- Easing: ease-in-out 권장
- 60fps 유지

After Effects → Lottie 내보내기 방법`,
			},
			"design_accessibility_check": {
				title: "접근성 검사",
				searches: []struct{ name, q string }{
					{"WCAG 접근성 기준", "WCAG 2.1 accessibility checklist design"},
					{"접근성 디자인 방법", "accessible design color contrast keyboard navigation"},
				},
				prompt: `다음 정보를 바탕으로 접근성 검사 체크리스트를 작성해줘.
%s
형식:
[색상 대비 (Color Contrast)]
- AA 기준: 4.5:1 (일반 텍스트)
- AA 기준: 3:1 (대형 텍스트 18px+)
- 검사 도구: WebAIM Contrast Checker

[키보드 내비게이션]
- Tab 순서 논리적 구성
- Focus 표시 가시성
- Skip Navigation 링크

[스크린 리더 지원]
- Alt 텍스트 작성 규칙
- ARIA 레이블 사용법
- 의미 있는 HTML 구조

[터치 타겟 크기]
- 최소 44×44px (iOS/Android)

자동 검사 도구: axe/WAVE/Lighthouse`,
			},
			"design_responsive_test": {
				title: "반응형 테스트",
				searches: []struct{ name, q string }{
					{"반응형 디자인 기준", "responsive design breakpoints best practice 2025"},
					{"모바일 퍼스트 디자인", "mobile first design testing checklist"},
				},
				prompt: `다음 정보를 바탕으로 반응형 디자인 테스트 가이드를 작성해줘.
%s
형식:
[브레이크포인트 기준]
| 디바이스 | 너비 | 기준 |
모바일: 375px~767px
태블릿: 768px~1279px
데스크톱: 1280px+
와이드: 1920px+

[테스트 체크리스트]
- 텍스트 가독성 (최소 16px)
- 이미지 비율 유지
- 터치 영역 크기
- 내비게이션 변환 (햄버거 메뉴)
- 테이블/차트 스크롤 처리

[테스트 도구]
Chrome DevTools / Responsively App / BrowserStack`,
			},
			"design_client_presentation": {
				title: "클라이언트 발표 자료",
				searches: []struct{ name, q string }{
					{"클라이언트 디자인 발표", "client design presentation best practice feedback"},
					{"디자인 설명 방법", "design rationale presentation storytelling"},
				},
				prompt: `다음 정보를 바탕으로 클라이언트 발표용 자료 구성 가이드를 작성해줘.
%s
형식:
[발표 구성 (20분 기준)]
0-3분: 프로젝트 목표 재확인
3-8분: 리서치 & 인사이트
8-18분: 디자인 시안 발표
18-20분: 다음 단계 제안

[시안 설명 방법]
- 왜 이 방향인가? (근거)
- 사용자에게 어떤 경험?
- 브랜드 가이드 부합 여부

[피드백 수렴 방법]
- 구체적 질문 3개 준비
- 주관적 의견 vs 사실 분리

[발표 후 처리]
- 피드백 정리 → 다음 버전 일정 제안`,
			},
			"design_portfolio_update": {
				title: "포트폴리오 업데이트",
				searches: []struct{ name, q string }{
					{"디자인 포트폴리오 구성", "design portfolio best practice 2025"},
					{"포트폴리오 케이스 스터디", "UX design portfolio case study structure"},
				},
				prompt: `다음 정보를 바탕으로 포트폴리오 업데이트 가이드를 작성해줘.
%s
형식:
[포트폴리오 구성 원칙]
- 작품 수: 3-5개 (적지만 깊게)
- 각 케이스 스터디 구성:
  1. 문제 정의 (Challenge)
  2. 내 역할 (My Role)
  3. 프로세스 (Research→Ideate→Design→Test)
  4. 결과물 (Final Design)
  5. 임팩트 (Impact/Result)

[플랫폼별 포트폴리오]
- Behance: 비주얼 중심
- Notion: 프로세스 중심
- 개인 웹사이트: 통합

[업데이트 루틴]
- 프로젝트 완료 직후 정리
- 분기 1회 전체 업데이트`,
			},
			// ── 프리랜서 (20개) ──────────────────────────────────
			"freelancer_client_manage": {
				title: "클라이언트 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 클라이언트 관리", "freelancer client management CRM tool 2025"},
					{"클라이언트 관계 유지", "client relationship management freelance"},
				},
				prompt: `다음 정보를 바탕으로 클라이언트 관리 시스템을 작성해줘.
%s
형식:
[클라이언트 DB 구조]
| 클라이언트 | 업종 | 담당자 | 프로젝트 | 마지막 연락 | 상태 | 다음 액션 |

[클라이언트 등급 분류]
A급: 반복 의뢰 / 고단가 / 빠른 결정
B급: 가끔 의뢰 / 보통 단가
C급: 일회성 / 저단가

[관계 유지 루틴]
- 분기 1회 근황 체크 메시지
- 프로젝트 완료 후 1개월 후속 연락
- 생일/명절 인사 (고급 클라이언트)

[미팅 알림 자동화 방법]`,
			},
			"freelancer_estimate": {
				title: "견적서 자동 생성",
				searches: []struct{ name, q string }{
					{"프리랜서 견적 기준", "프리랜서 견적서 작성 기준 단가 2025"},
					{"프로젝트 단가 시세", query + " 프리랜서 단가 시세 시장가"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 견적서 초안을 작성해줘.
%s
형식:
[견적서]
견적번호: 2025-001
유효기간: 발행일로부터 14일

| 항목 | 내용 | 단가 | 수량 | 금액 |
기획/설계
디자인/개발
수정 (N회 포함)
추가 수정: 별도 협의

소계: ___원
부가세(10%): ___원
합계: ___원

계약금(50%): 계약시
잔금(50%): 납품시

[시장 단가 기준]
[견적 이메일 발송 문구]`,
			},
			"freelancer_invoice": {
				title: "청구서 / 세금계산서 발행",
				searches: []struct{ name, q string }{
					{"프리랜서 청구서 발행", "freelancer invoice template tax 2025 Korea"},
					{"세금계산서 발행 방법", "전자 세금계산서 발행 방법 2025"},
				},
				prompt: `다음 정보를 바탕으로 청구서 및 세금계산서 발행 가이드를 작성해줘.
%s
형식:
[청구서 구성]
- 공급자 정보 (사업자등록번호 필수)
- 공급받는자 정보
- 공급 내역 (작업 항목/기간/금액)
- 공급가액 / 세액 / 합계

[세금계산서 발행 방법]
- 홈택스 전자세금계산서 발행 절차
- 발행 기한: 다음달 10일까지
- 지연 발행 시 가산세

[청구 이메일 문구]
[미수금 발생 시 대응 방법]`,
			},
			"freelancer_tax": {
				title: "세금/회계 정리",
				searches: []struct{ name, q string }{
					{"프리랜서 세금 정리", "프리랜서 종합소득세 절세 방법 2025"},
					{"1인사업자 경비 처리", "1인 사업자 경비 인정 항목 2025"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 세금/회계 정리 가이드를 작성해줘.
%s
형식:
[연간 세금 일정]
1월: 부가세 확정신고 (7~12월분)
5월: 종합소득세 신고
7월: 부가세 예정신고 (1~6월분)

[경비 처리 가능 항목]
✅ 확실: 사무용품/통신비/교통비/교육비
⚠️ 조건부: 식비(업무 목적)/차량유지비
❌ 불가: 개인 생활비

[수입·지출 분류 엑셀 구조]
날짜 / 내용 / 분류 / 금액 / 증빙

[절세 팁 TOP 5]`,
			},
			"freelancer_time_track": {
				title: "프로젝트 시간 추적",
				searches: []struct{ name, q string }{
					{"프리랜서 시간 관리", "freelancer time tracking productivity tool 2025"},
				},
				prompt: `다음 정보를 바탕으로 프로젝트 시간 추적 시스템을 작성해줘.
%s
형식:
[일일 작업 로그 포맷]
날짜:
프로젝트:
| 시간 | 작업 내용 | 소요시간 | 누적 |

[시간 추적 도구 추천]
- Toggl Track (무료/간편)
- Clockify (무료/팀 기능)
- Harvest (유료/청구 연동)

[프로젝트별 수익성 계산]
총 작업시간 × 시간당 단가 = 실제 수익
목표 단가 vs 실제 단가 비교

[시간 기록이 중요한 이유]
- 견적 정확도 향상
- 저수익 프로젝트 식별
- 클라이언트 보고 근거`,
			},
			"freelancer_portfolio": {
				title: "포트폴리오 업데이트",
				searches: []struct{ name, q string }{
					{"프리랜서 포트폴리오", "freelancer portfolio best practice 2025"},
					{"포트폴리오 플랫폼", "freelancer portfolio platform Behance LinkedIn"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 포트폴리오 업데이트 가이드를 작성해줘.
%s
형식:
[포트폴리오 핵심 원칙]
- 최근 작업 3-5개 집중
- 결과/임팩트 수치 포함 (매출 N% 증가 등)
- 클라이언트 추천사 포함

[플랫폼별 전략]
- LinkedIn: 전문성/신뢰도 중심
- Behance/Dribbble: 비주얼 중심
- 개인 사이트: 통합 브랜딩

[프로젝트 케이스 스터디 포맷]
1. 클라이언트/업종 (익명 가능)
2. 과제 (Challenge)
3. 솔루션
4. 결과 (임팩트)

[업데이트 주기]
프로젝트 완료 후 2주 이내 정리`,
			},
			"freelancer_self_marketing": {
				title: "자기 PR 콘텐츠 생성",
				searches: []struct{ name, q string }{
					{"프리랜서 자기 PR", "freelancer self marketing personal brand 2025"},
					{"LinkedIn 포스팅 전략", "LinkedIn content strategy freelancer thought leader"},
				},
				prompt: `다음 정보를 바탕으로 자기 PR 콘텐츠를 생성해줘.
%s
형식:
💼 LinkedIn 포스트 (전문가 버전)
[최근 작업 스토리: 300자]

📝 블로그/브런치 아티클
- 제목 3가지:
- 서론 초안:

🧵 X/스레드 포스트
- 10개 불릿 포인트:

📱 인스타그램 캡션
- 작업 과정 공유 버전:

[홍보 주기 추천]
LinkedIn: 주 2-3회
블로그: 월 2회
SNS: 주 3-4회`,
			},
			"freelancer_contract_review": {
				title: "계약서 검토",
				searches: []struct{ name, q string }{
					{"프리랜서 계약서 위험 항목", "freelancer contract red flags review 2025"},
					{"계약서 필수 조항", "freelance contract essential clauses Korea"},
				},
				prompt: `다음 정보를 바탕으로 프리랜서 계약서 검토 결과를 작성해줘.
%s
형식:
🔴 즉시 수정 요청 항목
- 무제한 수정 조항 (횟수 명시 필요)
- 지식재산권 과도한 양도
- 일방적 계약 해지 조건
- 무기한 비밀유지 조항

🟡 협상 권장 항목
- 납기 지연 패널티 기준
- 추가 작업 단가 기준
- 완료 기준(Acceptance Criteria)

🟢 확인 필수
- 계약금 비율 (최소 30%)
- 저작권 귀속 시점
- 분쟁 해결 방법

계약서 협상 팁 3가지 포함`,
			},
			"freelancer_cashflow": {
				title: "현금 흐름 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 현금 흐름", "freelancer cash flow management income stability"},
					{"수입 안정화 방법", "freelance income stabilization retainer contract"},
				},
				prompt: `다음 정보를 바탕으로 현금 흐름 관리 가이드를 작성해줘.
%s
형식:
[월별 현금 흐름 예측표]
| 월 | 예상 수입 | 예상 지출 | 잔액 |

[수입 안정화 전략]
1. 리테이너 계약 비율 목표: 40%
2. 프로젝트 다각화 (클라이언트 3개 이상)
3. 비상금 목표: 3개월치 생활비

[지출 분류]
고정: 통신비/구독/세금
변동: 외주/장비/교육

[미수금 방지 전략]
- 계약금 50% 선납
- 단계별 지급 구조
- 자동 청구 도구 설정`,
			},
			"freelancer_tax_report": {
				title: "연말정산 / 부가세 신고 자료",
				searches: []struct{ name, q string }{
					{"프리랜서 세금 신고", "프리랜서 종합소득세 신고 방법 2025"},
					{"부가세 신고 준비", "1인사업자 부가세 신고 준비 서류 2025"},
				},
				prompt: `다음 정보를 바탕으로 세금 신고 자료 정리 가이드를 작성해줘.
%s
형식:
[종합소득세 신고 준비 (5월)]
필요 서류:
- 수입 내역 (세금계산서/거래명세서)
- 경비 영수증 (카드/현금)
- 사업소득 원천징수영수증

[경비 정리 방법]
카테고리별 합산:
사무용품 / 통신비 / 교육비 / 차량 / 기타

[절세 체크리스트]
- 노란우산공제 가입 여부
- 청년우대형 계좌 활용
- 경비 누락 항목 확인

[신고 일정]
[홈택스 신고 절차 요약]`,
			},
			"freelancer_client_onboarding": {
				title: "클라이언트 온보딩",
				searches: []struct{ name, q string }{
					{"클라이언트 온보딩 방법", "freelancer client onboarding process template"},
				},
				prompt: `다음 정보를 바탕으로 신규 클라이언트 온보딩 패키지를 작성해줘.
%s
형식:
[온보딩 체크리스트]
계약 전:
□ 범위 정의 (SOW)
□ 견적 확정
□ 계약서 서명
□ 계약금 수령

계약 후:
□ 웰컴 메시지 발송
□ 킥오프 미팅 일정 확정
□ 협업 툴 초대 (Slack/Notion)
□ 자료 수집 (브리핑/에셋)

[웰컴 메시지 템플릿]
[킥오프 미팅 어젠다]
[초기 자료 요청 체크리스트]`,
			},
			"freelancer_project_kickoff": {
				title: "프로젝트 킥오프",
				searches: []struct{ name, q string }{
					{"프로젝트 킥오프 방법", "project kickoff meeting agenda template"},
					{"킥오프 미팅 구성", "kickoff meeting checklist freelancer"},
				},
				prompt: `다음 정보를 바탕으로 프로젝트 킥오프 자료를 작성해줘.
%s
형식:
[킥오프 미팅 어젠다 (60분)]
10분: 소개 및 아이스브레이킹
15분: 프로젝트 목표 재확인
15분: 범위 및 일정 확인
10분: 협업 방식 결정
10분: 질문 및 다음 단계

[킥오프 후 배포 문서]
- 프로젝트 요약
- 마일스톤 및 납기
- 연락처 및 에스컬레이션
- 협업 툴 링크

[프로젝트 계획서 1페이지 요약]`,
			},
			"freelancer_deliverable_check": {
				title: "산출물 검토",
				searches: []struct{ name, q string }{
					{"산출물 검토 체크리스트", "deliverable review checklist quality control freelance"},
				},
				prompt: `다음 정보를 바탕으로 납품 전 산출물 검토 체크리스트를 작성해줘.
%s
형식:
[납품 전 최종 체크리스트]

📁 파일 구성
□ 최종 파일 + 수정 가능 소스 파일
□ 파일명 규칙 준수
□ 폴더 구조 정리

✅ 품질 확인
□ 계약서의 납품 기준 충족
□ 수정 횟수 내 처리 완료
□ 오타/오류 최종 검수

📧 납품 이메일
- 납품 파일 안내
- 사용 방법 가이드
- 수정 정책 안내
- 잔금 청구 안내`,
			},
			"freelancer_payment_reminder": {
				title: "미수금 독촉",
				searches: []struct{ name, q string }{
					{"미수금 독촉 방법", "freelancer overdue invoice reminder email template"},
					{"미수금 법적 대응", "unpaid invoice freelance legal action Korea"},
				},
				prompt: `다음 정보를 바탕으로 미수금 독촉 메시지를 작성해줘.
%s
형식:
[D+3 (납기 3일 초과) - 정중한 리마인드]
제목: [프로젝트명] 대금 납부 안내
본문: 친절하고 간결하게

[D+14 - 공식 독촉]
제목: [프로젝트명] 미납 대금 독촉장
본문: 공식 톤 + 납부 기한 명시

[D+30 - 법적 대응 예고]
제목: 내용증명 발송 예정 안내
본문: 진지한 경고 톤

[법적 대응 절차]
1. 내용증명 → 2. 지급명령 → 3. 소액심판

[미수금 예방 방법 5가지]`,
			},
			"freelancer_proposal_template": {
				title: "제안서 템플릿 관리",
				searches: []struct{ name, q string }{
					{"프리랜서 제안서 구조", "freelancer proposal template winning 2025"},
					{"업종별 제안서 차이", "proposal structure design development marketing"},
				},
				prompt: `다음 정보를 바탕으로 업종별 제안서 템플릿을 작성해줘.
%s
형식:
[공통 제안서 구조]
1. 커버페이지 (클라이언트명 + 프로젝트명)
2. 우리의 이해 (클라이언트 상황/문제)
3. 제안 솔루션
4. 작업 범위 및 프로세스
5. 타임라인
6. 견적
7. 포트폴리오/레퍼런스
8. 계약 조건

[업종별 강조 포인트]
디자인: 비주얼 레퍼런스 풍부하게
개발: 기술 스택/아키텍처 명시
마케팅: 예상 ROI/성과 지표

[제안서 발송 후 팔로업 가이드]`,
			},
			"freelancer_rate_calculation": {
				title: "단가 계산",
				searches: []struct{ name, q string }{
					{"프리랜서 적정 단가", "freelancer rate calculation 2025 Korea"},
					{"업종별 프리랜서 단가", query + " 프리랜서 시장 단가 시세"},
				},
				prompt: `다음 정보를 바탕으로 적정 단가를 계산해줘.
%s
형식:
[시간당 단가 역산 계산]
목표 월 수입: ___원
실제 작업 가능 시간: ___시간/월 (총 근무시간 × 0.6)
→ 시간당 최소 단가: ___원

[프로젝트 단가 계산]
예상 작업시간 × 시간당 단가 = 기본 견적
+ 복잡도 가산 (1.2~1.5배)
+ 급행 가산 (1.3~2.0배)

[시장 단가 벤치마크]
업종별 평균 단가 (시간당/프로젝트)

[단가 인상 방법 및 타이밍]`,
			},
			"freelancer_work_log": {
				title: "작업 로그 정리",
				searches: []struct{ name, q string }{
					{"작업 로그 관리", "work log management freelancer productivity"},
				},
				prompt: `다음 정보를 바탕으로 오늘의 작업 로그 정리 가이드를 작성해줘.
%s
형식:
[일일 작업 로그]
📅 날짜:
🎯 오늘 목표:

| 시간 | 프로젝트 | 작업 내용 | 결과물 | 시간 |
09:00~
10:00~
...

✅ 완료한 작업
⚠️ 미완료 및 이유
📋 내일 이어서 할 것

[주간 작업 요약]
총 작업시간: ___h
프로젝트별 시간 배분:
수익성 체크:`,
			},
			"freelancer_business_plan": {
				title: "사업 계획 수립",
				searches: []struct{ name, q string }{
					{"1인 사업 계획", "freelance business plan 2025 growth strategy"},
					{"프리랜서 수익화 전략", "freelancer income growth strategy productize service"},
				},
				prompt: `다음 정보를 바탕으로 1인 사업 계획서를 작성해줘.
%s
형식:
1. 사업 비전 및 목표
2. 서비스 포지셔닝 (전문 분야 정의)
3. 타겟 고객 정의 (ICP)
4. 수익 모델 설계
   - 프로젝트형 / 리테이너형 / 디지털 제품
5. 연간 매출 목표 및 달성 전략
6. 마케팅/영업 계획
7. 역량 개발 계획
8. 리스크 및 대응 방안`,
			},
			"freelancer_networking_content": {
				title: "네트워킹 콘텐츠",
				searches: []struct{ name, q string }{
					{"프리랜서 네트워킹", "freelancer networking LinkedIn content strategy"},
					{"커뮤니티 활동 방법", "professional networking community freelance"},
				},
				prompt: `다음 정보를 바탕으로 네트워킹용 LinkedIn 콘텐츠를 작성해줘.
%s
형식:
[LinkedIn 포스트 3가지]

1. 인사이트 공유형 (전문성 어필)
훅: [첫 줄로 멈추게 하는 문장]
본문: [3-5개 핵심 포인트]
CTA: [댓글 유도]

2. 경험 스토리형 (공감 유발)
상황 → 시도 → 결과 → 교훈

3. 질문형 (커뮤니티 참여 유도)
[업계 공통 고민 던지기]

[네트워킹 DM 템플릿]
- 첫 연락 / 팔로업 / 협업 제안

[커뮤니티 추천]`,
			},
			"freelancer_yearly_review": {
				title: "연간 리뷰",
				searches: []struct{ name, q string }{
					{"프리랜서 연간 리뷰", "freelancer annual review reflection 2025"},
					{"1인 사업 성과 분석", "solo business year review growth metrics"},
				},
				prompt: `다음 정보를 바탕으로 연간 리뷰 리포트를 작성해줘.
%s
형식:
📊 연간 실적 요약
- 총 매출: / 목표 대비:
- 프로젝트 수: / 클라이언트 수:
- 평균 프로젝트 단가:

💼 클라이언트 분석
- 최고 수익 클라이언트 TOP 3
- 재계약율:
- 신규 vs 기존 비율:

🌱 성장 & 학습
- 올해 새로 배운 기술:
- 업그레이드된 역량:

⚠️ 아쉬운 점 & 교훈

🎯 내년 목표 설정
매출 목표 / 서비스 방향 / 역량 개발`,
			},
			// ── PM ──────────────────────────────────────────────
		}

		wfSelected, ok := presetDefs[preset]
		if !ok {
			wfErrMsg := "알 수 없는 워크플로우: " + preset
			if cx.req.Lang == "en" || isEnglishQuery(cx.req.Message) {
				wfErrMsg = "Unknown workflow: " + preset
			}
			json200(cx.w, CommandResponse{Success: false, Message: wfErrMsg, Action: "workflow_preset", Duration: cx.dur})
			return
		}

		for i, s := range wfSelected.searches {
			go func(idx int, name, q string) {
				tr, ok := tavilySearch(cx.tKey, q, 3)
				if ok {
					wfCh <- wfSection{name, tr.Summary}
				} else {
					wfCh <- wfSection{name, ""}
				}
			}(i, s.name, s.q)
		}

		wfCollected := []string{}
		for range wfSelected.searches {
			s := <-wfCh
			if s.body != "" {
				wfCollected = append(wfCollected, fmt.Sprintf("### %s\n%s", s.name, s.body))
			}
		}

		searchContext := strings.Join(wfCollected, "\n\n")
		finalPrompt := fmt.Sprintf(wfSelected.prompt, searchContext)
		if cx.req.Message != "" {
			if cx.req.Lang == "en" || isEnglishQuery(cx.req.Message) {
				finalPrompt = fmt.Sprintf("## User Request/Code\n%s\n\n%s", cx.req.Message, finalPrompt)
			} else {
				finalPrompt = fmt.Sprintf("## 사용자 요청/코드\n%s\n\n%s", cx.req.Message, finalPrompt)
			}
		}
		persona := getActivePersona()
		var wfSys string
		if cx.req.Lang == "en" || isEnglishQuery(cx.req.Message) {
			wfSys = persona.SystemPrompt + "\nAnswer in clear English using markdown formatting."
		} else {
			wfSys = persona.SystemPrompt + "\n답변은 마크다운으로 깔끔하게 작성하세요."
		}
		wfMsgs := []groqMsg{{Role: "system", Content: wfSys}, {Role: "user", Content: finalPrompt}}
		result, _, _ := callGroqWithFallback(wfMsgs, 1500, false)
		if result == "" {
			result, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: finalPrompt}}, 1500, false)
		}

		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  result,
			Action:   "workflow_preset",
			Result:   map[string]any{"preset": preset, "title": wfSelected.title, "persona": persona.ID},
			Duration: cx.dur,
		})

}

func cmdWorkflowPlan(cx cmdCtx) {
		var goal string
		if cx.params != nil {
			goal, _ = cx.params["goal"].(string)
		}
		if goal == "" {
			goal = cx.req.Message
		}
		// Reflection Loop: /api/workflow/run으로 내부 위임
		wfReqBody, _ := json.Marshal(map[string]any{"goal": goal, "use_reflection": true})
		wfResp, wfErr := (&http.Client{Timeout: 120 * time.Second}).Post(
			"http://127.0.0.1:17891/api/workflow/run", "application/json",
			bytes.NewReader(wfReqBody),
		)
		if wfErr == nil && wfResp != nil {
			var wfResult map[string]any
			json.NewDecoder(wfResp.Body).Decode(&wfResult)
			wfResp.Body.Close()
			summary, _ := wfResult["summary"].(string)
			if summary == "" {
				summary = fmt.Sprintf("'%s' 워크플로우 완료", goal)
			}
			json200(cx.w, CommandResponse{
				Success:  true,
				Message:  summary,
				Action:   "workflow_plan",
				Result:   wfResult,
				Duration: cx.dur,
			})
			return
		}
		// fallback: LLM 계획만 반환
		wfEng := isEnglishQuery(goal)
		var wfSys, wfUser string
		if wfEng {
			wfSys = "You are Jarvis AI. Write a step-by-step completion report for the given goal in English."
			wfUser = "Goal: " + goal
		} else {
			wfSys = "당신은 자비스 AI입니다. 주어진 목표를 단계별로 실행 완료 보고 형식으로 작성하세요."
			wfUser = "목표: " + goal
		}
		wMsgs := []groqMsg{
			{Role: "system", Content: wfSys},
			{Role: "user", Content: wfUser},
		}
		plan, _, _ := callGroqWithFallback(wMsgs, 800, false)
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  plan,
			Action:   "workflow_plan",
			Duration: cx.dur,
		})

}

func cmdMultiAction(cx cmdCtx) {
		subAction, _ := cx.params["sub_action"].(string)
		query, _ := cx.params["query"].(string)
		site, _ := cx.params["site"].(string)
		platform, _ := cx.params["platform"].(string)
		fmtStr, _ := cx.params["format"].(string)
		// pending_params의 format을 우선 사용 (LLM이 덮어쓰는 것 방지)
		if pf, ok := cx.req.PendingParams["format"].(string); ok && pf != "" {
			fmtStr = pf
		}
		maxItemsF, _ := cx.params["max_items"].(float64)
		maxItems := int(maxItemsF)
		if maxItems == 0 {
			maxItems = 8
		}
		if query == "" {
			query = cx.req.Message
		}
		outputFmt := outputFormat(fmtStr)

		llmMu.RLock()
		localTKey := llmTavilyKey; cx.tKey = localTKey
		llmMu.RUnlock()

		var collectedItems []map[string]string
		var actionSummary string

		switch subAction {
		case "price_compare":
			if cx.tKey != "" {
				if site != "" {
					if tr, ok := tavilySearchDomain(cx.tKey, query, maxItems, site); ok {
						collectedItems = tr.Items
					}
				}
				if len(collectedItems) == 0 {
					if tr, ok := tavilySearch(cx.tKey, query, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			siteName := site
			if siteName == "" {
				siteName = "쇼핑몰"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 상품 %d개 검색 결과", siteName, query, len(collectedItems))

		case "video_search":
			targetDomain := "youtube.com"
			if platform == "tiktok" {
				targetDomain = "tiktok.com"
			}
			if cx.tKey != "" {
				if tr, ok := tavilySearchDomain(cx.tKey, query, maxItems, targetDomain); ok {
					collectedItems = tr.Items
				}
				if len(collectedItems) == 0 {
					fallbackQ := query + " " + targetDomain
					if tr, ok := tavilySearch(cx.tKey, fallbackQ, maxItems); ok {
						collectedItems = tr.Items
					}
				}
			}
			pName := "YouTube"
			if platform == "tiktok" {
				pName = "TikTok"
			}
			actionSummary = fmt.Sprintf("%s에서 \"%s\" 영상 %d개 검색 결과", pName, query, len(collectedItems))

		case "doc_compare":
			// 두 대상 비교 - Tavily 검색 후 LLM이 비교표 생성
			llmMu.RLock()
			localGKey := llmPerplexityKey; cx.gKey = localGKey
			llmMu.RUnlock()
			docEng := cx.req.Lang == "en" || isEnglishQuery(query)
			var compareText string
			if tr, ok := webSearchWithFallback(cx.tKey, query, maxItems); ok {
				collectedItems = tr.Items
				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]
					c := item["content"]
					if c == "" { c = item["snippet"] }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
				}
				if tr.Summary != "" {
					if docEng {
						articleLines.WriteString("\n[Full Summary]\n" + tr.Summary)
					} else {
						articleLines.WriteString("\n[전체 요약]\n" + tr.Summary)
					}
				}
				var prompt string
				if docEng {
					prompt = fmt.Sprintf(`Based on the following information, compare "%s" by category.
Use a markdown comparison table (| Category | A | B |) format in English.

Reference material:
%s`, query, articleLines.String())
				} else {
					prompt = fmt.Sprintf(`다음 정보를 바탕으로 "%s"를 항목별로 비교 정리해줘.
마크다운 비교표(| 항목 | A | B |) 형식으로 한국어로 작성해줘.

참고 자료:
%s`, query, articleLines.String())
				}
				if cx.gKey != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": prompt}},
					}
					compareText = callClaudeAPI(llmClaudeKey, body)
				}
			}
			if compareText == "" {
				llmMu.RLock()
				gFallback := llmPerplexityKey
				cFallback := llmClaudeKey
				llmMu.RUnlock()
				var fallbackPrompt string
				if docEng {
					fallbackPrompt = fmt.Sprintf(`Compare "%s" by category using a markdown comparison table (| Category | A | B |) in English. Base it on the latest information.`, query)
				} else {
					fallbackPrompt = fmt.Sprintf(`"%s"를 항목별로 비교 정리해줘.
마크다운 비교표(| 항목 | A | B |) 형식으로 한국어로 작성해줘. 최신 정보를 기반으로 작성해줘.`, query)
				}
				if gFallback != "" {
					compareText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: fallbackPrompt}}, 2000, false)
				} else if cFallback != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": fallbackPrompt}},
					}
					compareText = callClaudeAPI(cFallback, body)
				}
				if compareText == "" {
					if docEng {
						compareText = fmt.Sprintf("Search quota exceeded for \"%s\". Add a Tavily API key in Settings to enable real-time comparison.", query)
					} else {
						compareText = fmt.Sprintf("\"%s\" — 검색 API 쿼터를 초과했습니다. 설정에서 Tavily API 키를 등록하면 실시간 데이터로 비교할 수 있습니다.", query)
					}
				}
			}
			actionSummary = compareText

		case "summarize":
			// 주제 요약 - Tavily 검색 후 실제 기사 본문 포함해서 LLM 요약
			llmMu.RLock()
			localGKey := llmPerplexityKey; cx.gKey = localGKey
			llmMu.RUnlock()
			sumEng := cx.req.Lang == "en" || isEnglishQuery(query)
			var summaryText string
			if tr, ok := webSearchWithFallback(cx.tKey, query, maxItems); ok {
				collectedItems = tr.Items

				var articleLines strings.Builder
				for i, item := range tr.Items {
					t := item["title"]
					c := item["content"]
					if c == "" { c = item["snippet"] }
					if t == "" { continue }
					articleLines.WriteString(fmt.Sprintf("[%d] %s\n", i+1, t))
					if c != "" { articleLines.WriteString(c + "\n") }
					articleLines.WriteString("\n")
				}
				if tr.Summary != "" {
					if sumEng {
						articleLines.WriteString("\n[Full Summary]\n" + tr.Summary)
					} else {
						articleLines.WriteString("\n[전체 요약]\n" + tr.Summary)
					}
				}

				var prompt string
				if sumEng {
					prompt = fmt.Sprintf(`Based on the following search results, clearly summarize "%s" in English.
Structure the key content by category (## subtitle, - bullet points). Do not include source URLs.

Search results:
%s`, query, articleLines.String())
				} else {
					prompt = fmt.Sprintf(`다음 검색 결과를 바탕으로 "%s"에 대해 한국어로 명확하게 요약 정리해줘.
핵심 내용을 항목별(## 소제목, - 포인트)로 구조화해서 작성해줘. 출처 URL은 포함하지 마.

검색 결과:
%s`, query, articleLines.String())
				}
				if cx.gKey != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 2000, false)
				} else if llmClaudeKey != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 2000,
						"messages": []map[string]any{{"role": "user", "content": prompt}},
					}
					summaryText = callClaudeAPI(llmClaudeKey, body)
				}
			}
			if summaryText == "" {
				llmMu.RLock()
				gFallback := llmPerplexityKey
				cFallback := llmClaudeKey
				llmMu.RUnlock()
				var fallbackPrompt string
				if sumEng {
					fallbackPrompt = fmt.Sprintf(`Summarize "%s" in English with structured sections (## subtitle, - bullet points). Base it on the latest information.`, query)
				} else {
					fallbackPrompt = fmt.Sprintf(`"%s"에 대해 한국어로 핵심 내용을 항목별(## 소제목, - 포인트)로 요약 정리해줘. 최신 동향을 기반으로 작성해줘.`, query)
				}
				if gFallback != "" {
					summaryText, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: fallbackPrompt}}, 1500, false)
				} else if cFallback != "" {
					body := map[string]any{
						"model": claudeHaikuModel, "max_tokens": 1500,
						"messages": []map[string]any{{"role": "user", "content": fallbackPrompt}},
					}
					summaryText = callClaudeAPI(cFallback, body)
				}
				if summaryText == "" {
					if sumEng {
						summaryText = fmt.Sprintf("Search quota exceeded for \"%s\". Add a Tavily API key in Settings to enable real-time summaries.", query)
					} else {
						summaryText = fmt.Sprintf("\"%s\" — 검색 API 쿼터를 초과했습니다. 설정에서 Tavily API 키를 등록하면 실시간 요약을 사용할 수 있습니다.", query)
					}
				}
			}
			actionSummary = summaryText

		default:
			// 일반 web_search — 본문 포함 수집
			llmMu.RLock()
			gKey2 := llmPerplexityKey
			llmMu.RUnlock()
			wsEng := cx.req.Lang == "en" || isEnglishQuery(query)
			if tr, ok := webSearchWithFallback(cx.tKey, query, maxItems); ok {
				collectedItems = tr.Items
				if gKey2 != "" {
					var lines strings.Builder
					for i, item := range tr.Items {
						t := item["title"]
						c := item["content"]
						if c == "" { c = item["snippet"] }
						lines.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, t, c))
					}
					if tr.Summary != "" {
						if wsEng {
							lines.WriteString("\n[Full Summary]\n" + tr.Summary)
						} else {
							lines.WriteString("\n[전체 요약]\n" + tr.Summary)
						}
					}
					var prompt string
					if wsEng {
						prompt = fmt.Sprintf(`Summarize the search results for "%s" in 3-5 key sentences in English.\n\n%s`, query, lines.String())
					} else {
						prompt = fmt.Sprintf(`"%s" 검색 결과를 한국어로 3~5줄 핵심 요약해줘.\n\n%s`, query, lines.String())
					}
					actionSummary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 800, false)
				}
			}
			if actionSummary == "" {
				if wsEng {
					actionSummary = fmt.Sprintf("%d search results for \"%s\"", len(collectedItems), query)
				} else {
					actionSummary = fmt.Sprintf("\"%s\" 검색 결과 %d개", query, len(collectedItems))
				}
			}
		}

		// format 기본값: markdown
		if outputFmt == "" {
			outputFmt = outMarkdown
		}

		// 파일 저장
		title := query
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20])
		}
		filePath, saveErr := saveResultToFile(outputFmt, title, collectedItems, actionSummary)
		var fileMsg string
		maEng := cx.req.Lang == "en" || isEnglishQuery(query)
		if saveErr != nil {
			if maEng {
				fileMsg = fmt.Sprintf("⚠️ File save failed: %s", saveErr.Error())
			} else {
				fileMsg = fmt.Sprintf("⚠️ 파일 저장 실패: %s", saveErr.Error())
			}
		} else {
			extMap := map[outputFormat]string{
				outPDF: "HTML(PDF용)", outWord: "DOCX", outExcel: "XLSX",
				outPowerPoint: "PPTX", outMarkdown: "MARKDOWN", outTXT: "TXT",
			}
			ext := extMap[outputFmt]
			if ext == "" { ext = strings.ToUpper(string(outputFmt)) }
			if maEng {
				fileMsg = fmt.Sprintf("📄 Saved as %s: %s", ext, filePath)
			} else {
				fileMsg = fmt.Sprintf("📄 %s 파일로 저장됨: %s", ext, filePath)
			}
		}

		resultItems := make([]map[string]string, 0, len(collectedItems))
		for _, it := range collectedItems {
			resultItems = append(resultItems, map[string]string{
				"site": site, "name": it["title"], "price": it["price"], "link": it["url"],
			})
		}

		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  actionSummary + "\n" + fileMsg,
			Action:   "multi_action",
			Result: map[string]any{
				"query":     query,
				"summary":   actionSummary,
				"results":   resultItems,
				"total":     len(resultItems),
				"file_path": filePath,
				"file_msg":  fileMsg,
				"format":    fmtStr,
				"sub_action": subAction,
			},
			Duration: cx.dur,
		})

}

