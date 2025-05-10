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
const widgetCSS = `
    @font-face {
        font-family: 'GowunDodum';
        src: url('https://fastly.jsdelivr.net/gh/projectnoonnu/noonfonts_2108@1.1/GowunDodum-Regular.woff') format('woff');
        font-weight: normal;
        font-style: normal;
        font-display: swap;
    }
    #webchat-widget-container,
    #webchat-widget-container * {
        font-family: 'GowunDodum', sans-serif !important;
        margin-top: 0 !important;
        margin-bottom: 0 !important;
        margin-left: 0 !important;
        margin-right: 0 !important;
        text-align: left !important;
    }
    #webchat-widget-container {
        position: fixed;
        bottom: 20px;
        right: 20px;
        z-index: 1000;
        max-width: 100vw;
        max-height: 100vh;
    }
    .chatbot-widget {
        background: white;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        width: 350px;
        height: 70vh;
        max-width: 95vw;
        max-height: 80vh;
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
        padding: 1rem;
        margin: 0 auto;
        font-family: 'GowunDodum', sans-serif !important;
        color: black;
        font-size: 16px;
    }
    .chatbot-title, .chatbot-desc {
        margin: 0.5rem 0;
    }
    .chatbot-messages {
        flex: 1;
        overflow-y: auto;
        margin-bottom: 10px;
        border: 1px solid #ccc;
        border-radius: 8px;
        padding: 1rem;
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
        gap: 10px;
    }
    .chatbot-input {
        flex: 1;
        padding: 8px;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 16px !important;
        background: #fafafa !important;
        color: black !important;
    }
    .chatbot-send-btn {
        padding: 8px 16px;
        background: #2e8b57;
        color: #fff;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }
    .chatbot-clear-btn {
        background-color: #e57373;
        color: #fff;
        border: none;
        font-size: 1rem;
        cursor: pointer;
        padding: 8px 12px;
        border-radius: 4px;
        // margin-left: 8px;
        transition: background-color 0.2s;
    }
    .chatbot-clear-btn:hover {
        background-color: #ef5350;
    }
    .chatbot-toggle-btn {
        position: fixed;
        bottom: 20px;
        right: 20px;
        width: 50px;
        height: 50px;
        border-radius: 50%;
        background: #2e8b57;
        color: #fff;
        border: none;
        font-size: 24px;
        cursor: pointer;
        z-index: 1001;
        box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        transition: bottom 0.3s ease;
    }
    @media (max-width: 600px) {
        #webchat-widget-container {
            bottom: 0;
            right: 0;
            left: 0;
            width: 100vw;
            max-width: 100vw;
            display: flex;
            justify-content: center;
            align-items: flex-end;
            overflow-x: hidden;
        }
        .chatbot-widget {
            width: 92vw !important;
            max-width: 380px !important;
            min-width: 0 !important;
            height: 70vh !important;
            max-height: 80vh !important;
            border-radius: 16px !important;
            padding: 8px !important;
            margin: 0 auto 8px auto !important;
            box-sizing: border-box !important;
            overflow-x: hidden !important;
        }
        .chatbot-toggle-btn {
            bottom: 16px;
            right: 16px;
            width: 48px;
            height: 48px;
            font-size: 22px;
        }
    }
`;

const style = document.createElement('style');
style.textContent = widgetCSS;
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
        styleTag.textContent = widgetCSS;
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