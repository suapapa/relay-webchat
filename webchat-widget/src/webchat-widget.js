(function() {
    // 위젯 스타일과 마크업을 동적으로 추가
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

    // 위젯 컨테이너 생성
    const container = document.createElement('div');
    container.id = 'webchat-widget-container';
    document.body.appendChild(container);

    // React 앱 초기화
    const root = document.createElement('div');
    root.id = 'webchat-root';
    container.appendChild(root);
    
    // React 앱이 로드될 때까지 대기
    const checkRender = setInterval(() => {
        if (typeof window.renderWebchatWidget === 'function') {
            clearInterval(checkRender);
            window.renderWebchatWidget(root);
        }
    }, 100);
})(); 