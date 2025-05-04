import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'

const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement)
root.render(
  <React.StrictMode>
    <App apiUrl="https://homin.dev/webchat-relay/chat" />
  </React.StrictMode>
)