# AGENTS.md — relay-webchat

homin.dev 웹챗 위젯용 3티어 스택. AI 에이전트·기여자용 규칙입니다.

## 프로젝트 한 줄

브라우저 위젯 → 클라우드 Relay → 로컬 Processor(AI/RAG)로 메시지를 중계합니다.

```
WebChat Widget  --REST-->  Relay  <--WS-->  Processor
```

## 읽기 순서

1. [`README.md`](README.md)
2. [`relay/.env.example`](relay/.env.example), [`processor/.env.example`](processor/.env.example) — 컴포넌트별 설정 SSOT
3. `webchat-widget/` — React 위젯
4. `relay/` — Gin + WebSocket 릴레이
5. `processor/` — 로컬 대화·RAG 처리 (`go_ragkit` + OpenAI 호환 API 사용)

## 레이아웃

| 경로 | 책임 |
|------|------|
| `webchat-widget/` | React UMD 위젯 (`npm run build` → `dist/`) |
| `relay/` | 퍼블릭 쪽 중계 서버 |
| `processor/` | 로컬 AI/검색 프로세서 |
| `relay.dockerfile` / `processor.dockerfile` | 컨테이너 이미지 |

## Go / 의존성

- Go **1.27** (`relay/`, `processor/` 각각 모듈).
- Processor는 `github.com/suapapa/go_ragkit` 헬퍼와 `github.com/openai/openai-go`를 사용합니다.
  - `NewPGVectorOpenAIVectorStore` + `DefaultOAIEmbedModel()` **함수** 호출 (변수 아님).
  - env: `RELAY_WS`, `PGVECTOR_CONN`, `PGVECTOR_TABLE_PHRASES`, `OPENAI_ENDPOINT`, `OPENAI_APIKEY`, `OPENAI_EMBED_MODEL`, `OPENAI_EMBED_DIM`, `OPENAI_CHAT_MODEL`.
- Relay/Processor 공통: `RELAY_APIKEY` (WS Bearer). Relay는 `ROOT_PATH`도 사용.
- 로컬 go_ragkit를 쓰려면 `processor/go.mod`에 `replace => ../../go_ragkit` 유지.

## 환경변수

### Relay (`relay/.env.example`)

| 변수 | 용도 |
|------|------|
| `RELAY_APIKEY` | Relay `/ws` ↔ Processor Bearer (비어 있으면 인증 비활성) |
| `ROOT_PATH` | Relay Gin root path |
| `ALLOWED_ORIGINS` | CORS 허용 오리진 (콤마 구분, `*`는 전체 허용) |
| `RATE_LIMIT_RPM` | `/chat` IP별 분당 요청 한도 (기본 10, 0은 비활성화) |
| `RATE_LIMIT_BURST` | `/chat` IP별 버스트 허용량 (기본 5) |

### Processor (`processor/.env.example`)

| 변수 | 용도 |
|------|------|
| `RELAY_WS` | 접속할 Relay WebSocket 주소 (`-ws` 플래그 기본값, 기본 `ws://localhost:8080`) |
| `RELAY_APIKEY` | Relay `/ws` ↔ Processor Bearer (비어 있으면 인증 비활성) |
| `PGVECTOR_CONN` | PostgreSQL/pgvector DSN |
| `PGVECTOR_TABLE_PHRASES` | phrases 테이블명 |
| `OPENAI_ENDPOINT` / `OPENAI_APIKEY` | OpenAI 호환 API 엔드포인트 및 키 (임베딩·채팅 공통) |
| `OPENAI_EMBED_MODEL` | 임베딩 모델 (기본 `text-embedding-3-small`) |
| `OPENAI_EMBED_DIM` | 임베딩 차원 (기본 `1536`, 모델·기존 테이블 컬럼 차원과 일치해야 함) |
| `OPENAI_CHAT_MODEL` | LLM 채팅(전처리) 모델 (기본 `gpt-4o-mini`) |

API 키는 env만 사용. `/secret/token` 파일 경로를 되살리지 말 것.

## 빌드 산출물

다음 바이너리는 커밋하지 않습니다 (`.gitignore`):

- `relay/relay`
- `processor/processor`, `processor/processor_*`

위젯은 `webchat-widget/dist/` 도 무시합니다. 배포 산출물은 CI/스크립트로 복사.

## 코딩 규칙

- Relay와 Processor의 WebSocket 프로토콜/메시지 스키마를 한쪽만 바꾸지 말 것.
- 시크릿은 env; 저장소에 토큰 파일을 넣지 말 것.
- Weaviate/`WEAVIATE_*` / `OPENAI_SECRET_KEY` / Ollama / Genkit(Gemini, `GEMINI_API_KEY`) 경로를 재도입하지 말 것 (명시 요청 없이).
- `deploy.sh`의 호스트·아키텍처는 로컬 환경 의존 — 임의로 공용 기본값으로 덮어쓰지 말 것.
- 위젯 공개 API(UMD 글로벌·마운트 방식) 변경 시 README 삽입 예제를 갱신.

## 하지 말 것

- `.DS_Store`, 빌드 바이너리, `node_modules`, `.env` 커밋
- 사용자 요청 없이 커밋·푸시

## 커밋

사용자 요청 시에만. 메시지 스타일: `feat|fix|chore: …`
