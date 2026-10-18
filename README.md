## Relay-WebChat

![relay-webchat logo](_asset/relay-webchat_logo.webp)

이 프로젝트는 개인 사이트, homin.dev 의 웹쳇 위젯을 로컬 서버에서 호스팅 하기 위해
세 개의 컴포넌트로 구성되어 있습니다.

```
┌─────────────────┐        ┌─────────────┐      ┌─────────────┐
│  WebChat Widget │◀─rest─▶│    Relay    │◀─ws─▶│  Processor  │
│    (client)     │        │   (cloud)   │      │  (local AI) │
└─────────────────┘        └─────────────┘      └─────────────┘
```

## WebChat Widget

React로 작성되어 있는 웹쳇 위젯.

Build:
```sh
cd webchat-widget
npm install
npm run build
```

웹페이지에 추가하기:
1. `dist/webchat-widget.umd.js`를 적당한 곳에 호스팅.
2. `dist/index.html` 파일을 참고해 웹쳇 위젯을 추가할 페이지에 다음의 예제와 같이 추가.

```html
</body>
  ...
  <script src="https://example.com/asset/script/webchat-widget.umd.js"></script>
</body>
```

## Relay

퍼블릭 클라우드에서 호스팅 중인, Go로 작성된 채팅 릴레이 서버.

웹쳇 위젯으로 부터 받은 메세지를 웹소켓으로 연결되어 있는 프로세서 서버에 전달하여 처리 결과를 응답.

## Processor

로컬 서버에서 호스팅 하는, Go로 작성된 대화 처리 서버.

pgvector + OpenAI 호환 임베딩으로 블로그 검색을 수행하고, Genkit(Gemini)로 전처리를 할 수 있습니다.

설정은 [`.env.example`](.env.example)을 참고하세요 (`RELAY_APIKEY`, `PGVECTOR_CONN`, `OPENAI_*`).

## Container images (GHCR)

native amd64/arm64 (QEMU 없음):

- `ghcr.io/suapapa/relay-webchat/relay`
- `ghcr.io/suapapa/relay-webchat/processor`