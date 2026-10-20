# Relay-WebChat

![relay-webchat logo](_asset/relay-webchat_logo.webp)

개인 사이트([homin.dev](https://homin.dev))의 블로그 검색 웹챗 봇(**블검봇**)을 로컬 서버에서 호스팅하기 위한 3티어(Widget · Relay · Processor) 프로젝트입니다.

브라우저의 웹챗 위젯이 퍼블릭 클라우드의 **Relay**로 REST 요청을 보내면, Relay와 상시 WebSocket으로 연결된 로컬 **Processor**가 LLM 전처리 및 RAG(pgvector) 검색을 수행해 응답을 돌려줍니다.

```
┌─────────────────┐        ┌─────────────┐      ┌─────────────┐
│  WebChat Widget │◀─REST─▶│    Relay    │◀─WS─▶│  Processor  │
│    (client)     │        │   (cloud)   │      │  (local AI) │
└─────────────────┘        └─────────────┘      └─────────────┘
```

## WebChat Widget

React + TypeScript(Vite)로 작성된 UMD 웹챗 위젯입니다. Shadow DOM으로 호스트 페이지와 스타일을 격리하며, `localStorage` 대화 내역 보존과 마크다운 응답 렌더링을 지원합니다.

Build:
```sh
cd webchat-widget
npm install
npm run build
```

웹페이지에 추가하기:
1. 빌드 산출물인 `dist/webchat-widget.umd.js`를 정적 호스팅 경로에 배포합니다.
2. 위젯을 표시할 페이지에 아래와 같이 스크립트를 추가하면 자동으로 우측 하단에 마운트됩니다 (`dist/index.html` 참고).

```html
<body>
  ...
  <!-- data-api-url 생략 시 기본값: https://homin.dev/webchat-relay/chat -->
  <script
    src="https://example.com/asset/script/webchat-widget.umd.js"
    data-api-url="https://homin.dev/webchat-relay/chat"
  ></script>
</body>
```

수동으로 특정 컨테이너에 마운트하려면 전역 함수 `window.renderWebchatWidget(container, { apiUrl })`을 호출할 수 있습니다.

## Relay

퍼블릭 클라우드에서 호스팅하는 Go(Gin) 기반 채팅 릴레이 서버입니다.

- `POST <ROOT_PATH>/chat`: 웹챗 위젯의 메시지를 받아 연결된 Processor로 전달하고 응답을 반환합니다 (IP별 토큰 버킷 Rate Limit 및 CORS 설정 지원).
- `GET <ROOT_PATH>/ws`: Processor가 접속하는 WebSocket 엔드포인트입니다 (`RELAY_APIKEY` 설정 시 Bearer 토큰 인증).

환경변수 설정은 [`relay/.env.example`](relay/.env.example)을 참고하세요:

| 환경변수 | 플래그 | 기본값 | 설명 |
|----------|--------|--------|------|
| `RELAY_APIKEY` | - | `""` | `/ws` 접속용 공유 Bearer 토큰 (비어 있으면 인증 비활성) |
| `ADDR` | `-addr` | `:8080` | 서버 리슨 주소 |
| `ROOT_PATH` | `-root` | `/` | 라우트 루트 경로 |
| `ALLOWED_ORIGINS` | `-origins` | `https://homin.dev,...` | CORS 허용 오리진 (콤마 구분, `*`는 전체 허용) |
| `RATE_LIMIT_RPM` | `-rate-limit-rpm` | `10` | `/chat` IP별 분당 요청 한도 (`0`은 비활성화) |
| `RATE_LIMIT_BURST` | `-rate-limit-burst` | `5` | `/chat` IP별 버스트 허용량 |

## Processor

로컬 서버에서 호스팅하는 Go 기반 대화·RAG 처리 데몬입니다.

Relay의 `/ws`에 연결(지수 백오프 자동 재연결 및 주기적 Ping/Pong)하여 질문을 수신하고, OpenAI 호환 API로 사용자 의도·키워드를 전처리(`/keyword`, `/smallchat`, `/about`)한 뒤 `go_ragkit` + PostgreSQL(`pgvector`)로 블로그 포스트 임베딩 유사도 검색을 수행합니다.

환경변수 설정은 [`processor/.env.example`](processor/.env.example)을 참고하세요:

| 환경변수 | 플래그 | 기본값 | 설명 |
|----------|--------|--------|------|
| `RELAY_WS` | `-ws` | `ws://localhost:8080` | 접속할 Relay WebSocket 주소 |
| `RELAY_APIKEY` | - | `""` | Relay `/ws` 공유 Bearer 토큰 |
| `PGVECTOR_CONN` | - | (필수) | PostgreSQL/pgvector 연결 DSN |
| `PGVECTOR_TABLE_PHRASES` | - | `homin_dev_phrases_openai` | 검색할 phrases 벡터 테이블명 |
| `OPENAI_ENDPOINT` | - | `https://api.openai.com/v1` | OpenAI 호환 API 엔드포인트 (임베딩·채팅 공통) |
| `OPENAI_APIKEY` | - | (필수) | OpenAI 호환 API 키 |
| `OPENAI_EMBED_MODEL` | - | `text-embedding-3-small` | 임베딩 모델명 |
| `OPENAI_EMBED_DIM` | - | `1536` | 임베딩 벡터 차원 (모델 및 테이블 컬럼 차원과 일치 필요) |
| `OPENAI_CHAT_MODEL` | - | `gpt-4o-mini` | 의도 분류/전처리용 채팅 모델명 |
| - | `-pre-process` | `false` | LLM 기반 프롬프트 전처리(의도·키워드 추출) 활성화 |
| - | `-retrive` | `50` | 벡터 DB에서 검색할 청크(문구) 개수 |

실행 예시:
```sh
cd processor
cp .env.example .env
# .env 편집 후 로컬 실행
./run.sh -pre-process -retrive 10

# 또는 Docker Compose 실행
docker compose up -d
```

## Container images (GHCR)

GitHub Actions를 통해 멀티 아키텍처(`linux/amd64`, `linux/arm64`) 네이티브 이미지가 빌드·배포됩니다:

- `ghcr.io/suapapa/relay-webchat/relay`
- `ghcr.io/suapapa/relay-webchat/processor`

