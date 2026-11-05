import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './i18n/index'
import './index.css'

const root = document.getElementById('root') // 前端应用的挂载节点
if (!root) throw new Error('Missing application root')

// 在严格模式中挂载应用，便于开发阶段发现不安全的副作用
ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
