import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

// 전역 window 객체에 renderWebchatWidget 함수 타입 선언
declare global {
    interface Window {
        renderWebchatWidget: (container?: HTMLElement) => void;
    }
}

// 스타일 동적 추가
const style = document.createElement('style');
style.textContent = `
    #webchat-widget-container {
        position: fixed;
        bottom: 20px;
        right: 20px;
        z-index: 1000;
    }
    .chatbot-widget {
        background: white;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        padding: 20px;
        width: 300px;
        height: 400px;
        display: flex;
        flex-direction: column;
    }
    .chatbot-messages {
        flex: 1;
        overflow-y: auto;
        margin-bottom: 10px;
    }
    .chatbot-input-row {
        display: flex;
        gap: 10px;
    }
    .chatbot-input {
        flex: 1;
        padding: 8px;
        border: 1px solid #ddd;
        border-radius: 4px;
    }
    .chatbot-send-btn {
        padding: 8px 16px;
        background: #007bff;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }
    .chatbot-toggle-btn {
        position: fixed;
        bottom: 20px;
        right: 20px;
        width: 50px;
        height: 50px;
        border-radius: 50%;
        background: #007bff;
        color: white;
        border: none;
        font-size: 24px;
        cursor: pointer;
        z-index: 1001;
        transition: bottom 0.3s ease;
    }
`;
document.head.appendChild(style);

// 컨테이너 생성 (ShadowRoot 지원)
function ensureWidgetContainerWithShadow() {
    let container = document.getElementById('webchat-widget-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'webchat-widget-container';
        document.body.appendChild(container);
    }
    // ShadowRoot 생성
    let shadowRoot = (container as any).shadowRoot;
    if (!shadowRoot) {
        shadowRoot = container.attachShadow({ mode: 'open' });
    }
    // ShadowRoot 내부에 root div 생성
    let root = shadowRoot.getElementById('webchat-root');
    if (!root) {
        root = document.createElement('div');
        root.id = 'webchat-root';
        shadowRoot.appendChild(root);
    }
    // App.css를 ShadowRoot에 style로 삽입
    if (!shadowRoot.getElementById('webchat-style')) {
        const styleTag = document.createElement('style');
        styleTag.id = 'webchat-style';
        styleTag.textContent = `
.chatbot-widget-container {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
}

.chatbot-widget {
  width: 350px;
  height: 500px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 24px rgba(0,0,0,0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 2rem;
  font-family: Arial, sans-serif !important;
  color: black;
  font-size: 16px;
}

.chatbot-title {
  margin: 0;
  text-align: left;
  font-family: 'Arial', sans-serif !important;
  font-size: 1.5rem !important;
  font-weight: bold !important;
  color: #333 !important;
}

.chatbot-desc {
  margin-top: 0.5rem;
  text-align: left;
  font-family: 'Arial', sans-serif !important;
  font-size: 1rem !important;
  color: #666 !important;
}

.chatbot-messages {
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 1rem;
  height: 300px;
  overflow-y: auto;
  margin-bottom: 1rem;
  flex: 1;
  background: #fafafa;
}

.chatbot-message {
  margin: 0.5rem 0;
}

.chatbot-message.user {
  text-align: right;
}

.chatbot-message.bot {
  text-align: left;
}

.chatbot-input-row {
  display: flex;
  gap: 0.5rem;
}

.chatbot-input {
  flex: 1;
  padding: 0.5rem;
  background: #fafafa !important;
  color: black !important;
}

.chatbot-send-btn {
  padding: 0.5rem 1rem;
  background: #2e8b57;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.chatbot-toggle-btn {
  position: absolute;
  right: 0;
  border-radius: 50%;
  width: 56px;
  height: 56px;
  background: #2e8b57;
  color: #fff;
  border: none;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  cursor: pointer;
  font-size: 24px;
  transition: bottom 0.2s;
}
        `;
        shadowRoot.appendChild(styleTag);
    }
    return root;
}

// 전역 함수로 React 앱 렌더링 함수 노출 (ShadowRoot 지원)
window.renderWebchatWidget = (container?: HTMLElement) => {
    const target = container || ensureWidgetContainerWithShadow();
    const root = ReactDOM.createRoot(target);
    root.render(
        <React.StrictMode>
            <App />
        </React.StrictMode>
    );
}

// 자동 실행: 페이지에 위젯 컨테이너가 없으면 자동으로 렌더링
if (!document.getElementById('webchat-widget-container')) {
    window.renderWebchatWidget();
}