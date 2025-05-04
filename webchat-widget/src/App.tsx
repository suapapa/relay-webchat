import React, { useState } from 'react'
import axios from 'axios'
import './App.css'

interface Message {
  sender: 'user' | 'bot'
  text: string
}

function App({ apiUrl = 'https://homin.dev/webchat-relay/chat' }) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [open, setOpen] = useState(false)

  const sendMessage = async () => {
    if (!input.trim()) return

    const userMessage: Message = { sender: 'user', text: input }
    setMessages((prev) => [...prev, userMessage])

    try {
      const response = await axios.post(`${apiUrl}`, { message: input });
      const botMessage: Message = { sender: 'bot', text: response.data.reply || 'No response' };
      setMessages((prev) => [...prev, botMessage]);
    } catch (err) {
      const errorMsg: Message = { sender: 'bot', text: 'Error contacting backend.' }
      setMessages((prev) => [...prev, errorMsg])
    }

    setInput('')
  }

  return (
    <div>
      <div className="chatbot-widget-container">
        {open && (
          <div className="chatbot-widget">
            <h1 className="chatbot-title">Hello World!</h1>
            <p className="chatbot-desc">This is a custom chatbot using a local backend.</p>

            <div className="chatbot-messages">
              {messages.map((msg, idx) => (
                <div key={idx} className={msg.sender === 'user' ? 'chatbot-message user' : 'chatbot-message bot'}>
                  <strong>{msg.sender === 'user' ? 'You' : 'Bot'}:</strong> {msg.text}
                </div>
              ))}
            </div>

            <div className="chatbot-input-row">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
                    sendMessage();
                  }
                }}
                className="chatbot-input"
                placeholder="Type your message..."
              />
              <button onClick={sendMessage} className="chatbot-send-btn">Send</button>
            </div>
          </div>
        )}
        <button
          className="chatbot-toggle-btn"
          style={{ bottom: open ? '510px' : '24px' }}
          onClick={() => setOpen((o) => !o)}
        >
          {open ? '×' : '🍀'}
        </button>
      </div>
    </div>
  )
}

export default App