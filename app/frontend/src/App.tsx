import { useEffect, useRef, useState } from 'react'
import { OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime'
import { About, Sidebar, Toasts, useVersions, WorkspaceBar } from './components/Shell'
import { backend } from './lib/backend'
import { ConvertPage } from './pages/ConvertPage'
import { ExplorePage } from './pages/ExplorePage'
import { LintPage } from './pages/LintPage'
import { MergePage } from './pages/MergePage'
import { VerifyPage } from './pages/VerifyPage'
import { AppProvider, useApp } from './state'

function Shell() {
  const { page, open, inspect, zones, fail } = useApp()
  const versions = useVersions()
  const [about, setAbout] = useState(false)

  // The drop handler is registered once; it reads the current actions through a ref.
  const actions = useRef({ open, inspect, fail })
  actions.current = { open, inspect, fail }
  useEffect(() => {
    try {
      // A drop goes to the zone under the pointer when a page registered one
      // (data-drop); anywhere else in the window files it by what it is.
      OnFileDrop((x, y, paths) => {
        const zone = document.elementFromPoint(x, y)?.closest('[data-drop]')?.getAttribute('data-drop') ?? ''
        const handler = zones.current.get(zone)
        const { open, inspect, fail } = actions.current
        if (handler) inspect(paths).then(handler, fail)
        else void open(paths)
      }, true)
    } catch {
      // Outside the app window there is no native drop.
    }
    backend.startupFiles().then(paths => {
      if (paths && paths.length > 0) void actions.current.open(paths)
    }, () => {})
    return () => {
      try {
        OnFileDropOff()
      } catch {
        // As above.
      }
    }
  }, [zones])

  return (
    <div className="app">
      <Sidebar versions={versions} onAbout={() => setAbout(true)} />
      <div className="main">
        <WorkspaceBar />
        {/* Every page stays mounted, so a result survives a look at another page. */}
        <main className="page" hidden={page !== 'convert'}><ConvertPage /></main>
        <main className="page fill" hidden={page !== 'explore'}><ExplorePage active={page === 'explore'} /></main>
        <main className="page" hidden={page !== 'lint'}><LintPage /></main>
        <main className="page" hidden={page !== 'merge'}><MergePage /></main>
        <main className="page" hidden={page !== 'verify'}><VerifyPage /></main>
      </div>
      <Toasts />
      {about && <About versions={versions} onClose={() => setAbout(false)} />}
    </div>
  )
}

export default function App() {
  return (
    <AppProvider>
      <Shell />
    </AppProvider>
  )
}
