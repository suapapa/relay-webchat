import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './App.css'

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

// 컨테이너 생성
function ensureWidgetContainer() {
    let container = document.getElementById('webchat-widget-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'webchat-widget-container';
        document.body.appendChild(container);
    }
    let root = document.getElementById('webchat-root');
    if (!root) {
        root = document.createElement('div');
        root.id = 'webchat-root';
        container.appendChild(root);
    }
    return root;
}

// 전역 함수로 React 앱 렌더링 함수 노출
window.renderWebchatWidget = (container?: HTMLElement) => {
    const target = container || ensureWidgetContainer();
    const root = ReactDOM.createRoot(target)
    root.render(
        <React.StrictMode>
            <App />
        </React.StrictMode>
    )
}

// 자동 실행: 페이지에 위젯 컨테이너가 없으면 자동으로 렌더링
if (!document.getElementById('webchat-widget-container')) {
    window.renderWebchatWidget();
}