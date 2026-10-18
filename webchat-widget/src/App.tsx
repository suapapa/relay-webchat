import React, { useState, useRef, useEffect } from 'react'
import axios from 'axios'
import { marked } from 'marked'

interface Message {
  sender: 'user' | 'bot'
  text: string
  isError?: boolean
}

export interface AppProps {
  apiUrl?: string
}

const STORAGE_KEY = 'chatMessages'
export const DEFAULT_API_URL = 'https://homin.dev/webchat-relay/chat'

function loadSavedMessages(): Message[] {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (!saved) return []
    const parsed = JSON.parse(saved)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (m): m is Message =>
        m &&
        typeof m === 'object' &&
        (m.sender === 'user' || m.sender === 'bot') &&
        typeof m.text === 'string' &&
        !m.isError
    )
  } catch {
    return []
  }
}

function renderBotMarkdown(text: string): string {
  const rawHtml = marked.parse(text, { async: false }) as string
  const template = document.createElement('template')
  template.innerHTML = rawHtml

  template.content.querySelectorAll('script, iframe, object, embed').forEach((el) => el.remove())
  template.content.querySelectorAll('*').forEach((el) => {
    Array.from(el.attributes).forEach((attr) => {
      const name = attr.name.toLowerCase()
      const value = attr.value.trim().toLowerCase()
      if (name.startsWith('on') || ((name === 'href' || name === 'src') && value.startsWith('javascript:'))) {
        el.removeAttribute(attr.name)
      }
    })
  })

  return template.innerHTML
}

function App({ apiUrl = DEFAULT_API_URL }: AppProps) {
  const [messages, setMessages] = useState<Message[]>(loadSavedMessages)
  const [input, setInput] = useState('')
  const [open, setOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement | null>(null)
  const startInFlightRef = useRef(false)

  useEffect(() => {
    if (!open || messages.length > 0 || startInFlightRef.current) {
      return
    }

    let cancelled = false
    startInFlightRef.current = true
    setIsLoading(true)

    axios
      .post(apiUrl, { message: '/start' })
      .then((response) => {
        if (cancelled) return
        const botMessage: Message = {
          sender: 'bot',
          text: response.data?.reply || 'No response',
        }
        setMessages((prev) => (prev.length === 0 ? [botMessage] : prev))
      })
      .catch((err) => {
        if (cancelled) return
        const errorMsg: Message = {
          sender: 'bot',
          text: err?.response?.data?.reply || 'Error contacting backend.',
          isError: true,
        }
        setMessages((prev) => (prev.length === 0 ? [errorMsg] : prev))
      })
      .finally(() => {
        startInFlightRef.current = false
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [open, messages.length, apiUrl])

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages, open])

  useEffect(() => {
    try {
      const persistable = messages.filter((m) => !m.isError)
      if (persistable.length === 0) {
        localStorage.removeItem(STORAGE_KEY)
      } else {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(persistable))
      }
    } catch {
      // Ignore storage quota or private mode errors
    }
  }, [messages])

  const clearMessages = () => {
    setMessages([])
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // Ignore storage errors
    }
  }

  const sendMessage = async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return

    setIsLoading(true)
    const userMessage: Message = { sender: 'user', text: trimmed }
    setMessages((prev) => [...prev.filter((m) => !m.isError), userMessage])
    setInput('')

    try {
      const response = await axios.post(apiUrl, { message: trimmed })
      const botMessage: Message = {
        sender: 'bot',
        text: response.data?.reply || 'No response',
      }
      setMessages((prev) => [...prev, botMessage])
    } catch (err: any) {
      const errorMsg: Message = {
        sender: 'bot',
        text: err?.response?.data?.reply || 'Error contacting backend.',
        isError: true,
      }
      setMessages((prev) => [...prev, errorMsg])
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div>
      <div className="chatbot-widget-container">
        {open && (
          <div className="chatbot-widget">
            <h2 className="chatbot-title">🍀 블검봇</h2>
            <p className="chatbot-desc">GenAI chatbot to help you search Homin Lee's blog</p>

            <div className="chatbot-messages">
              {messages.map((msg, idx) => (
                <div key={idx} className={msg.sender === 'user' ? 'chatbot-message user' : 'chatbot-message bot'}>
                  <strong>{msg.sender === 'user' ? 'You' : 'Bot'}:</strong>{' '}
                  {msg.sender === 'user' ? (
                    <span>{msg.text}</span>
                  ) : (
                    <span dangerouslySetInnerHTML={{ __html: renderBotMarkdown(msg.text) }} />
                  )}
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            <div className="chatbot-input-row">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.nativeEvent.isComposing && !isLoading) {
                    e.preventDefault()
                    sendMessage()
                  }
                }}
                className="chatbot-input"
                placeholder="Type your message..."
                disabled={isLoading}
              />
              <button onClick={sendMessage} className="chatbot-send-btn" disabled={isLoading}>
                {isLoading ? 'Sending...' : 'Send'}
              </button>
              <button onClick={clearMessages} className="chatbot-clear-btn" title="Clear chat history">
                X
              </button>
            </div>
          </div>
        )}
        <button
          className="chatbot-toggle-btn"
          style={{ bottom: open ? '70vh' : '24px' }}
          onClick={() => setOpen((o) => !o)}
        >
          {open ? '×' : '🍀'}
        </button>
      </div>
    </div>
  )
}

export default App