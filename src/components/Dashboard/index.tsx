import { AnimatePresence, motion } from 'framer-motion'
import { useAppStore } from '../../stores/appStore'
import { Layout } from '../Layout'
import { Home } from './Home'
import { RepairView } from '../Repair'
import { SecurityView } from '../Security'
import { FileConverterView } from '../FileConverter'
import { TranslatorView } from '../Translator'
import { ClipboardView } from '../Clipboard'
import { MemoView } from '../Memo'
import { SettingsView } from '../Settings'
import { AutoCleanView } from '../AutoClean'
import { MonitorView } from '../Monitor'
import { PrivacyView } from '../Privacy'
import { FocusView } from '../FocusMode'
import { DailyView } from '../DailyOptimizer'
import { VoiceMemoView } from '../VoiceMemo'
import { SmartOrganizeView } from '../SmartOrganize'
import { PredictiveCareView } from '../PredictiveCare'

const pageVariants = {
  initial: { opacity: 0, y: 6 },
  animate: { opacity: 1, y: 0 },
  exit:    { opacity: 0, y: -6 },
}

function ViewContent() {
  const { currentView } = useAppStore()
  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={currentView}
        variants={pageVariants}
        initial="initial"
        animate="animate"
        exit="exit"
        transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
        style={{ display: 'flex', flex: 1, overflow: 'hidden', minHeight: 0 }}
      >
        {currentView === 'home'        && <Home />}
        {currentView === 'repair'      && <RepairView />}
        {currentView === 'security'    && <SecurityView />}
        {currentView === 'files'       && <FileConverterView />}
        {currentView === 'translate'   && <TranslatorView />}
        {currentView === 'clipboard'   && <ClipboardView />}
        {currentView === 'memo'        && <MemoView />}
        {currentView === 'settings'    && <SettingsView />}
        {currentView === 'autoclean'   && <AutoCleanView />}
        {currentView === 'monitor'     && <MonitorView />}
        {currentView === 'privacy'     && <PrivacyView />}
        {currentView === 'focus'       && <FocusView />}
        {currentView === 'daily'       && <DailyView />}
        {currentView === 'voicememo'   && <VoiceMemoView />}
        {currentView === 'organize'    && <SmartOrganizeView />}
        {currentView === 'predictive'  && <PredictiveCareView />}
      </motion.div>
    </AnimatePresence>
  )
}

export function Dashboard() {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}
    >
      <Layout>
        <ViewContent />
      </Layout>
    </motion.div>
  )
}
