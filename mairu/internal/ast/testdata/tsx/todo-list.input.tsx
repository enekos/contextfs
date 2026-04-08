import React, { useState } from 'react'

interface Todo {
  id: number
  text: string
  done: boolean
}

export function TodoList() {
  const [todos, setTodos] = useState<Todo[]>([])
  const [input, setInput] = useState('')

  function addTodo() {
    if (!input.trim()) return
    setTodos([...todos, { id: Date.now(), text: input, done: false }])
    setInput('')
  }

  function toggleTodo(id: number) {
    setTodos(todos.map(t => t.id === id ? { ...t, done: !t.done } : t))
  }

  function removeTodo(id: number) {
    setTodos(todos.filter(t => t.id !== id))
  }

  return (
    <div>
      <input value={input} onChange={e => setInput(e.target.value)} />
      <button onClick={addTodo}>Add</button>
      <ul>
        {todos.map(t => (
          <li key={t.id} style={{ textDecoration: t.done ? 'line-through' : 'none' }}>
            <span onClick={() => toggleTodo(t.id)}>{t.text}</span>
            <button onClick={() => removeTodo(t.id)}>x</button>
          </li>
        ))}
      </ul>
    </div>
  )
}
