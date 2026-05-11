import { useAppStore } from './stores/appStore'
import { FloatingCharacter } from './components/FloatingCharacter'
import './styles/design-system.css'

export default function App() {
  const { isOnboarded } = useAppStore()

  return (
    <div
      style={{
        width: '100vw',
        height: '100vh',
        background: 'rgba(0,0,0,0.95)',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <FloatingCharacter />
    </div>
  )
}
