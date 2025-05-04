import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    lib: {
      entry: 'src/index.tsx',
      name: 'WebchatWidget',
      fileName: 'webchat-widget',
      formats: ['umd'],
    },
    rollupOptions: {
      // React를 번들에 포함하려면 external을 비워둡니다.
      external: [],
      output: {
        globals: {
          react: 'React',
          'react-dom': 'ReactDOM',
        },
      },
    },
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'production'),
    'process.env': '{}', // 혹시 모를 다른 참조도 방지
    'process': '{}',     // 혹시 모를 다른 참조도 방지
  },
})