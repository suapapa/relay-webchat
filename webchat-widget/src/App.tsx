import React, { useState, useRef, useEffect } from 'react'
import axios from 'axios'
import { marked } from 'marked'

interface Message {
  sender: 'user' | 'bot'
  text: string
}

function App({ apiUrl = 'https://homin.dev/webchat-relay/chat' }) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [open, setOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages, open])

  const sendMessage = async () => {
    if (!input.trim() || isLoading) return

    setIsLoading(true)
    const userMessage: Message = { sender: 'user', text: input }
    setMessages((prev) => [...prev, userMessage])

    try {
      const response = await axios.post(`${apiUrl}`, { message: input });
      const botMessage: Message = { sender: 'bot', text: response.data.reply || 'No response' };
      setMessages((prev) => [...prev, botMessage]);
    } catch (err) {
      const errorMsg: Message = { sender: 'bot', text: 'Error contacting backend.' }
      setMessages((prev) => [...prev, errorMsg])
    } finally {
      setIsLoading(false)
      setInput('')
    }
  }

  return (
    <div>
      <div className="chatbot-widget-container">
        {open && (
          <div className="chatbot-widget">
            <h1 className="chatbot-title">🍀 블검봇</h1>
            <p className="chatbot-desc">Chatbot for searching Homin Lee's Blog</p>

            <div className="chatbot-messages">
              {messages.map((msg, idx) => (
                <div key={idx} className={msg.sender === 'user' ? 'chatbot-message user' : 'chatbot-message bot'}>
                  <strong>{msg.sender === 'user' ? 'You' : 'Bot'}:</strong> <span dangerouslySetInnerHTML={{ __html: marked(msg.text) }} />
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            <div className="chatbot-input-row">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onInput={(e) => setInput(e.currentTarget.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing && !isLoading) {
                    e.preventDefault();
                    sendMessage();
                  }
                }}
                className="chatbot-input"
                placeholder="Type your message..."
                disabled={isLoading}
              />
              <button onClick={sendMessage} className="chatbot-send-btn" disabled={isLoading}>
                {isLoading ? 'Sending...' : 'Send'}
              </button>
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