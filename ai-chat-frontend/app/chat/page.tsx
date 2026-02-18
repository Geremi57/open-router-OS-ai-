'use client'

import { useState, useRef, useEffect } from 'react'

export default function ChatPage() {
  const [messages, setMessages] = useState<{ role: string; content: string }[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(scrollToBottom, [messages])

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim()) return

    // Add user message
    setMessages((prev) => [...prev, { role: 'user', content: input }])
    setLoading(true)

    try {
      const res = await fetch("http://localhost:8080/chat", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: input }),
      })
      // console.log(message)
      const data = await res.json()

      // Add AI response
      setMessages((prev) => [...prev, { role: 'ai', content: data.reply }])
    } catch (err) {
      console.error(err)
      setMessages((prev) => [...prev, { role: 'ai', content: 'Error: could not reach AI' }])
    }

    setInput('')
    setLoading(false)
  }

  return (
    <div className="flex flex-col h-screen bg-background text-foreground p-4">
      <div className="flex-1 overflow-y-auto space-y-2">
        {messages.map((msg, i) => (
          <div key={i} className={msg.role === 'user' ? 'text-right' : 'text-left'}>
            <span className="inline-block p-2 rounded-md bg-card">{msg.content}</span>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSend} className="flex gap-2 mt-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="flex-1 p-2 rounded border border-border bg-background text-foreground"
          placeholder="Ask something..."
          disabled={loading}
        />
        <button
          type="submit"
          className="px-4 py-2 rounded bg-primary text-primary-foreground disabled:opacity-50"
          disabled={loading || !input.trim()}
        >
          {loading ? '...' : 'Send'}
        </button>
      </form>
    </div>
  )
}
