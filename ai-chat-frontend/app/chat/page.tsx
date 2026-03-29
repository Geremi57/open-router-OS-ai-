'use client'

import { useState, useRef, useEffect } from 'react'
import { Send, Sparkles, Bot, User, Loader2, Settings, Menu, Heart, BookOpen, Clock } from 'lucide-react'

interface Message {
  role: string
  content: string
  timestamp?: Date
}

interface UserInfo {
  id: string
  name: string
  conversationCount: number
  mood?: string
}

export default function ChatPage() {
  const [messages, setMessages] = useState<Message[]>([
    {
      role: 'ai',
      content: "Hello! I'm Space. I'm so glad to meet you! What's your name? ✨",
      timestamp: new Date()
    }
  ])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const [user, setUser] = useState<UserInfo | null>(null)
  const [showNamePrompt, setShowNamePrompt] = useState(true)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Load saved user from localStorage
  useEffect(() => {
    const savedUser = localStorage.getItem('space_user')
    if (savedUser) {
      const parsedUser = JSON.parse(savedUser)
      setUser(parsedUser)
      setShowNamePrompt(false)
      // Welcome back message would be handled by AI
    }
  }, [])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || loading) return

    const userMessage = input.trim()
    setInput('')
    
    // Add user message to UI
    setMessages(prev => [...prev, { 
      role: 'user', 
      content: userMessage,
      timestamp: new Date()
    }])
    setLoading(true)

    try {
      const payload: any = { input: userMessage }
      
      // Add user info if available

if (user) {
  payload.userId = user.id
} else {
  // Check localStorage directly as fallback
  const savedUser = localStorage.getItem('space_user')
  if (savedUser) {
    const parsed = JSON.parse(savedUser)
    payload.userId = parsed.id
  } else if (showNamePrompt && userMessage.length < 30) {
    payload.username = userMessage
  }
}


      const res = await fetch("http://localhost:8080/chat", {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)

      const data = await res.json()
      
      // Save user info if this is first interaction
      if (!user && data.userId) {
        const newUser = {
          id: data.userId,
          name: data.username || userMessage,
          conversationCount: 1
        }
        setUser(newUser)
        localStorage.setItem('space_user', JSON.stringify(newUser))
        setShowNamePrompt(false)
      } else if (user && data.userId) {
        // Update conversation count
        setUser(prev => prev ? { ...prev, conversationCount: data.conversationCount || prev.conversationCount } : prev)
      }
      
      setMessages(prev => [...prev, { 
        role: 'ai', 
        content: data.reply,
        timestamp: new Date()
      }])
    } catch (err) {
      console.error(err)
      setMessages(prev => [...prev, { 
        role: 'ai', 
        content: '💫 I\'m having trouble connecting right now. Could you try again?',
        timestamp: new Date()
      }])
    } finally {
      setLoading(false)
      inputRef.current?.focus()
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend(e)
    }
  }

  const clearChat = () => {
    if (confirm('Clear conversation? Space will remember you but the chat history will be reset.')) {
      setMessages([{
        role: 'ai',
        content: user ? `Welcome back ${user.name}! I've cleared our conversation, but I still remember you. How are you feeling today? 🌟` : "Hello! I'm Space. I'm so glad to meet you! What's your name? ✨",
        timestamp: new Date()
      }])
    }
  }

  const resetUser = () => {
    if (confirm('Reset memory? This will clear all memories about you.')) {
      localStorage.removeItem('space_user')
      setUser(null)
      setShowNamePrompt(true)
      setMessages([{
        role: 'ai',
        content: "Hello! I'm Space. It's like meeting for the first time! What's your name? ✨",
        timestamp: new Date()
      }])
    }
  }

  return (
    <div className="flex h-screen bg-gradient-to-br from-indigo-900 via-purple-900 to-pink-900">
      {/* Sidebar */}
      {isSidebarOpen && (
        <div className="fixed inset-0 bg-black/50 z-40 lg:hidden" onClick={() => setIsSidebarOpen(false)} />
      )}
      <div className={`
        fixed lg:static inset-y-0 left-0 z-50 w-96 bg-white/10 backdrop-blur-xl border-r border-white/20
        transform transition-transform duration-300 ease-in-out flex flex-col
        ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
      `}>
        <div className="p-6 flex-1 overflow-y-auto">
          <div className="flex items-center gap-3 mb-8">
            <div className="p-3 bg-gradient-to-br from-purple-500 to-pink-500 rounded-2xl">
              <Sparkles className="w-7 h-7 text-white" />
            </div>
            <div>
              <h1 className="text-white font-bold text-2xl">Space</h1>
              <p className="text-white/60 text-sm">Your thoughtful companion</p>
            </div>
          </div>

          {user && (
            <div className="mb-6 p-4 bg-white/10 rounded-2xl border border-white/10">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                  <User className="w-5 h-5 text-white" />
                </div>
                <div>
                  <p className="text-white font-semibold">{user.name}</p>
                  <p className="text-white/40 text-xs">Friend since today</p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-white/60 text-sm">
                <Heart className="w-4 h-4" />
                <span>{user.conversationCount} conversations together</span>
              </div>
            </div>
          )}

          <button
            onClick={clearChat}
            className="w-full py-3 px-4 bg-white/10 hover:bg-white/20 rounded-xl text-white text-sm font-medium transition-all duration-200 border border-white/10 mb-3"
          >
            Clear Conversation
          </button>

          {user && (
            <button
              onClick={resetUser}
              className="w-full py-3 px-4 bg-red-500/20 hover:bg-red-500/30 rounded-xl text-red-200 text-sm font-medium transition-all duration-200 border border-red-500/30"
            >
              Reset Memory
            </button>
          )}

          <div className="mt-8 pt-6 border-t border-white/10">
            <h3 className="text-white/40 text-xs uppercase tracking-wider font-semibold mb-3">About Space</h3>
            <p className="text-white/60 text-sm leading-relaxed">
              I'm designed to be a thoughtful companion who remembers our conversations and grows our friendship over time. I'm here to listen, support, and share meaningful moments with you.
            </p>
          </div>
        </div>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10 bg-white/5 backdrop-blur-sm">
          <button
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="lg:hidden p-2 hover:bg-white/10 rounded-lg transition-colors"
          >
            <Menu className="w-5 h-5 text-white" />
          </button>
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
            <span className="text-white/80 text-sm">Space is listening</span>
          </div>
          <div className="flex items-center gap-1 text-white/40 text-xs">
            <Clock className="w-3 h-3" />
            <span>Always here</span>
          </div>
        </div>

        {/* Messages Container */}
        <div className="flex-1 overflow-y-auto scroll-smooth">
          <div className="max-w-4xl mx-auto px-4 py-6 space-y-4">
            {messages.map((msg, i) => (
              <div
                key={i}
                className={`flex gap-3 animate-in fade-in slide-in-from-bottom-2 duration-300 ${
                  msg.role === 'user' ? 'justify-end' : 'justify-start'
                }`}
              >
                {msg.role === 'ai' && (
                  <div className="flex-shrink-0 w-9 h-9 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center shadow-lg">
                    <Bot className="w-5 h-5 text-white" />
                  </div>
                )}
                <div
                  className={`max-w-[80%] rounded-2xl px-5 py-3 shadow-lg ${
                    msg.role === 'user'
                      ? 'bg-gradient-to-r from-purple-500 to-pink-500 text-white'
                      : 'bg-white/10 backdrop-blur-sm text-white border border-white/10'
                  }`}
                >
                  <p className="whitespace-pre-wrap break-words leading-relaxed">{msg.content}</p>
                  {msg.timestamp && (
                    <p className="text-xs mt-1 opacity-50">
                      {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </p>
                  )}
                </div>
                {msg.role === 'user' && (
                  <div className="flex-shrink-0 w-9 h-9 rounded-full bg-white/10 flex items-center justify-center shadow-lg">
                    <User className="w-5 h-5 text-white" />
                  </div>
                )}
              </div>
            ))}
            {loading && (
              <div className="flex gap-3 justify-start">
                <div className="flex-shrink-0 w-9 h-9 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                  <Bot className="w-5 h-5 text-white" />
                </div>
                <div className="bg-white/10 backdrop-blur-sm rounded-2xl px-5 py-3 border border-white/10">
                  <div className="flex gap-1">
                    <div className="w-2 h-2 bg-white/60 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                    <div className="w-2 h-2 bg-white/60 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                    <div className="w-2 h-2 bg-white/60 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        </div>

        {/* Input Area */}
        <div className="border-t border-white/10 bg-white/5 backdrop-blur-sm p-4">
          <form onSubmit={handleSend} className="max-w-4xl mx-auto">
            <div className="relative flex items-center gap-2">
              <input
                ref={inputRef}
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                className="flex-1 p-4 pr-12 rounded-2xl bg-white/10 border border-white/20 text-white placeholder-white/40 focus:outline-none focus:ring-2 focus:ring-purple-500/50 focus:border-transparent transition-all duration-200"
                placeholder={user ? `Tell me something, ${user.name}...` : "What's your name? Or tell me anything..."}
                disabled={loading}
              />
              <button
                type="submit"
                disabled={loading || !input.trim()}
                className="absolute right-2 p-2 rounded-xl bg-gradient-to-r from-purple-500 to-pink-500 text-white disabled:opacity-50 disabled:cursor-not-allowed hover:shadow-lg transition-all duration-200"
              >
                {loading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <Send className="w-5 h-5" />
                )}
              </button>
            </div>
            <p className="text-white/30 text-xs text-center mt-3 flex items-center justify-center gap-2">
              <Heart className="w-3 h-3" />
              Space remembers our conversations
              <BookOpen className="w-3 h-3 ml-1" />
            </p>
          </form>
        </div>
      </div>
    </div>
  )
}