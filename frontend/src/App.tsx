import { Routes } from 'react-router-dom'
import { AppRoutes } from './routes'

export function App() {
  return (
    <main>
      <Routes>{AppRoutes()}</Routes>
    </main>
  )
}
