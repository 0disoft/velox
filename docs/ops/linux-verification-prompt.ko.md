# Velox Linux 검증 요청

첨부한 Velox 소스 ZIP을 실제 Linux 환경에서 검증해 주세요. 정적 코드 검토나
Linux 대상 크로스 컴파일만으로 끝내지 말고, 테스트와 집계 도구를 직접 실행하세요.

## 목적과 범위

- Velox는 현재 Windows x64 앱 패키저입니다. Linux 데스크톱 앱 지원을 추가하는
  작업이 아니라, Linux 기여자가 공통 Go 코드와 개발 도구를 검증할 수 있는지
  확인하는 작업입니다.
- 구현 기준 커밋은 `1e1be215b0cc979d46c0af5e3704a97f273c742d`입니다.
  그 이후 이 요청 문서만 추가해 ZIP을 만들었습니다.
- ZIP에는 `.git`, 캐시, 빌드 산출물, `node_modules`가 없습니다. 일부 커밋은
  아직 원격에 없으므로 GitHub main으로 소스를 바꿔치기하지 마세요.
- `third_party/go-webview2`와 스키마는 포함되어 있습니다. 저장소 루트에서
  실행하고, `go.mod`의 로컬 replace 경로를 그대로 유지하세요.
- `AGENTS.md`, `CONTRIBUTING.md`, `VALIDATION.md`와 관련 코드를 읽으세요.
  이 요청에서는 Mustflow나 명령 계약 등록 없이 일반 명령을 직접 실행합니다.

## 실행

1. Linux 배포판, 아키텍처, Go 버전과 네트워크 사용 가능 여부를 기록하세요.
   `uname -a`, `go version`, `go env GOOS GOARCH CGO_ENABLED`를 사용하세요.
   `go.mod`가 요구하는 Go 1.26.0 이상을 사용합니다. Windows 비교 환경은
   Go 1.27.1이었습니다. 버전이 다르면 명시하세요. Go가 없으면 공식 배포본을
   사용하고, 설치나 모듈 다운로드가 막히면 환경 제약으로 보고하세요.
2. 수정 전 원본 상태에서 아래 명령을 실행하고, 각 종료 코드와 전체 로그를
   보존하세요. 실패해도 이후 독립적인 검증은 계속하세요.

   ```sh
   go test -count=1 ./...
   go vet ./...
   go test -count=1 -v ./cmd/velox-consumer-summary
   ```

3. `cmd/velox-consumer-summary/main_test.go`의 유효한 관측 데이터 구조를 참고해
   합성 JSON 입력을 별도 작업 폴더에 만드세요. 기존 스키마와 필드를 유지하고,
   실제 측정 데이터가 아닌 합성 입력임을 결과에 명시하세요. 아래 명령을
   실제 Linux에서 실행해 JSON 출력과 종료 코드를 확인하세요.

   ```sh
   go run ./cmd/velox-consumer-summary --results-root linux-verification/raw --output linux-verification/summary.json --expected-samples 1
   ```

4. 정상 입력은 종료 코드 0과 요약 JSON, 기대 샘플 수를 2로 바꾼 실행은
   non-zero와 누락 샘플이 기록된 요약 JSON을 생성하는지 확인하세요.
   잘못된 JSON, 중복 ID, 스키마 위반, 실패 샘플, 서로 다른 릴리스 해시,
   hosted 증거의 프로세스 상태 처리는 위 Go 테스트 결과와 연결해 보고하세요.
5. Windows 전용 테스트가 빌드 태그로 제외되는 것은 정상입니다. WebView2,
   네이티브 대화상자, 트레이, 단축키, 실제 Windows 설치·실행은 미검증으로
   분리하세요. `./...`는 별도 Go 모듈의 테스트를 자동으로 실행하지 않습니다.
   Windows 실행파일을 Wine으로 실행하거나, 그 결과를 Linux 앱 지원으로
   해석하지 마세요. PowerShell을 설치해서 공통 도구 검증을 우회하지 마세요.

## 실패 처리와 반환물

- 오류를 환경 제약, Windows 전용 검증, 테스트의 OS 가정, 실제 공통 코드의
  이식성 결함으로 구분하세요. 실패를 무조건 skip하거나 테스트를 삭제하지 마세요.
- 실제 이식성 결함을 재현했다면 원본을 보존하고, 가장 작은 수정만 적용하세요.
  관련 테스트를 다시 실행하고 원본 기준 unified diff 패치를 함께 반환하세요.
  임의 기능 추가, 광범위 리팩토링, 의존성 업데이트, 버전 변경은 하지 마세요.
- GitHub 이슈·PR·푸시·태그·릴리스는 만들지 마세요. ZIP에는 Git 이력이 없으므로
  Git 메타데이터를 요구하는 작업은 억지로 실행하지 마세요.
- `linux-verification-report.md`에 환경, 실행 명령, 종료 코드, 통과·실패·제외
  범위, 원인과 파일/줄 위치, 수정 여부를 작성하세요. 각 검증을 실행 전의
  결과와 수정 후 결과로 구분하세요. 전체 환경 변수나 인증 정보를 출력하지 마세요.
- 보고서, 명령 로그, 입력·출력 JSON, 수정했다면 `.patch`를 하나의 ZIP 또는
  `tar.gz`로 묶어 돌려주세요. 실행하지 않은 항목은 실행했다고 쓰지 마세요.
