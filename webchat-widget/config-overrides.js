const path = require('path');

module.exports = function override(config, env) {
    // 단일 파일로 번들링
    config.output = {
        ...config.output,
        filename: 'webchat-widget.js',
        library: 'webchatWidget',
        libraryTarget: 'umd',
        globalObject: 'this'
    };

    // 엔트리 포인트 수정
    config.entry = path.resolve(__dirname, 'src/index.tsx');

    return config;
}; 