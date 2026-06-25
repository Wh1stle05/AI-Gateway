const messagesEl = document.getElementById('messages')
const inputEl = document.getElementById('input')
const sendBtn = document.getElementById('sendBtn')
const clearBtn = document.getElementById('clearBtn')
const modelSelect = document.getElementById('model')
const streamToggle = document.getElementById('streamToggle')
const statusEl = document.getElementById('status')
const statsEl = document.getElementById('stats')
const emptyEl = document.getElementById('empty')

let messages = []
let sending = false

const AVAILABLE_MODELS = ['gpt-4o-mini', 'gpt-4o', 'llama3.2', 'qwen2.5', 'mock-model']

// Init model select
AVAILABLE_MODELS.forEach(m => {
  const opt = document.createElement('option')
  opt.value = m
  opt.textContent = m
  modelSelect.appendChild(opt)
})

// Health check
async function checkHealth() {
  try {
    const res = await fetch('/health')
    statusEl.classList.toggle('ok', res.ok)
  } catch {
    statusEl.classList.remove('ok')
  }
}
checkHealth()
setInterval(checkHealth, 10000)

// Auto resize textarea
inputEl.addEventListener('input', () => {
  inputEl.style.height = 'auto'
  inputEl.style.height = Math.min(inputEl.scrollHeight, 150) + 'px'
})

// Send on Enter
inputEl.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
})

sendBtn.addEventListener('click', sendMessage)
clearBtn.addEventListener('click', clearChat)

function clearChat() {
  messages = []
  messagesEl.innerHTML = ''
  emptyEl.style.display = ''
  statsEl.textContent = ''
}

function addMessage(role, content, meta) {
  emptyEl.style.display = 'none'

  const div = document.createElement('div')
  div.className = `message ${role}`

  const label = document.createElement('div')
  label.className = 'message-label'
  label.textContent = role === 'user' ? '你' : 'AI'

  const bubble = document.createElement('div')
  bubble.className = 'message-bubble'
  bubble.textContent = content

  div.appendChild(label)
  div.appendChild(bubble)

  if (meta) {
    const metaEl = document.createElement('div')
    metaEl.className = 'message-meta'
    metaEl.textContent = meta
    div.appendChild(metaEl)
  }

  messagesEl.appendChild(div)
  messagesEl.scrollTop = messagesEl.scrollHeight
  return bubble
}

function addError(msg) {
  emptyEl.style.display = 'none'
  const div = document.createElement('div')
  div.className = 'message error'
  const label = document.createElement('div')
  label.className = 'message-label'
  label.textContent = 'Error'
  const bubble = document.createElement('div')
  bubble.className = 'message-bubble'
  bubble.textContent = msg
  div.appendChild(label)
  div.appendChild(bubble)
  messagesEl.appendChild(div)
  messagesEl.scrollTop = messagesEl.scrollHeight
}

async function sendMessage() {
  const text = inputEl.value.trim()
  if (!text || sending) return

  sending = true
  sendBtn.disabled = true
  inputEl.value = ''
  inputEl.style.height = 'auto'
  statsEl.textContent = ''

  messages.push({ role: 'user', content: text })
  addMessage('user', text)

  const model = modelSelect.value
  const stream = streamToggle.checked
  const start = Date.now()

  try {
    if (stream) {
      await streamResponse(model, start)
    } else {
      await jsonResponse(model, start)
    }
  } catch (err) {
    addError(err.message || '请求失败')
  } finally {
    sending = false
    sendBtn.disabled = false
    inputEl.focus()
  }
}

async function jsonResponse(model, start) {
  const res = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ model, messages }),
  })

  const data = await res.json()
  const elapsed = ((Date.now() - start) / 1000).toFixed(2)

  if (!res.ok) {
    addError(data.error?.message || `HTTP ${res.status}`)
    return
  }

  const content = data.choices?.[0]?.message?.content || ''
  const usage = data.usage
  let meta = `${elapsed}s`
  if (usage) {
    meta += ` | tokens: ${usage.total_tokens} (prompt: ${usage.prompt_tokens}, completion: ${usage.completion_tokens})`
  }

  messages.push({ role: 'assistant', content })
  addMessage('assistant', content, meta)
  statsEl.textContent = `耗时 ${elapsed}s` + (usage ? ` | 总 token: ${usage.total_tokens}` : '')
}

async function streamResponse(model, start) {
  const res = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ model, messages, stream: true }),
  })

  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    addError(data.error?.message || `HTTP ${res.status}`)
    return
  }

  const bubble = addMessage('assistant', '')
  let fullText = ''
  let tokenCount = 0

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop()

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed || !trimmed.startsWith('data:')) continue
      const data = trimmed.slice(5).trim()
      if (data === '[DONE]') continue

      try {
        const parsed = JSON.parse(data)
        const delta = parsed.choices?.[0]?.delta?.content
        if (delta) {
          fullText += delta
          tokenCount++
          bubble.textContent = fullText
          messagesEl.scrollTop = messagesEl.scrollHeight
        }
      } catch {}
    }
  }

  const elapsed = ((Date.now() - start) / 1000).toFixed(2)
  messages.push({ role: 'assistant', content: fullText })

  const metaEl = bubble.parentElement.querySelector('.message-meta') || (() => {
    const m = document.createElement('div')
    m.className = 'message-meta'
    bubble.parentElement.appendChild(m)
    return m
  })()
  metaEl.textContent = `${elapsed}s | ~${tokenCount} tokens (streaming)`

  statsEl.textContent = `耗时 ${elapsed}s | ~${tokenCount} tokens (streaming)`
}
