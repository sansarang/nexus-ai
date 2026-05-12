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
        background: 'transparent',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <FloatingCharacter />
    </div>
  )
}
