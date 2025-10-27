import Editor from '@monaco-editor/react'
import { useRef, useState, useEffect } from 'react'
import Login from './components/Login'
import api from './services/api'
import DiskList from './components/DiskList'
import FileSystemViewer from './components/FileSystemViewer'
import GraphView from './components/GraphView'
import Modal from './components/Modal'
import Navbar from './components/Navbar'

function App() {
  const editorRef = useRef(null)
  const consolaRef = useRef(null)

  const [ setEntradaFile ] = useState("")
  const [session, setSession] = useState(null)
  const [toast, setToast] = useState(null)

  useEffect(() => {
    if (!toast) return
    const t = setTimeout(() => setToast(null), 3000)
    return () => clearTimeout(t)
  }, [toast])
  const [selectedDisk, setSelectedDisk] = useState(null)
  const [view, setView] = useState('editor') // 'editor' | 'disks' | 'partitions' | 'explorer' | 'graph'
  const [showLoginModal, setShowLoginModal] = useState(false)

  const handleEditor = (editor, id) => {
    if (id == "editor") {
      editorRef.current = editor
    } else if (id == "consola") {
      consolaRef.current = editor
    }
  }

  // file selection handler (Navbar will call onFileSelected)
  const [uploadedFile, setUploadedFile] = useState(null)
  const handleFileSelected = (f) => {
    if (!f) return
    setUploadedFile(f)
    console.log('App: Selected file for upload:', f.name, f.size)

    // Read file as text and paste into the Entrada editor if possible
    const reader = new FileReader()
    reader.onload = (ev) => {
      const text = ev.target.result
      // try to set the editor content now; if editor not yet ready, retry a few times
      const trySet = () => {
        try {
          if (editorRef && editorRef.current) {
            const ed = editorRef.current
            if (typeof ed.setValue === 'function') {
              ed.setValue(text)
              console.log('App: File content pasted into editor via setValue')
              return true
            }
            if (ed.getModel && typeof ed.getModel === 'function') {
              const model = ed.getModel()
              if (model && typeof model.setValue === 'function') {
                model.setValue(text)
                console.log('App: File content pasted into editor via model.setValue')
                return true
              }
            }
            console.warn('App: Editor instance does not support setValue')
            return false
          }
          console.warn('App: Editor reference not ready, will retry shortly')
          return false
        } catch (err) {
          console.error('App: Failed to set editor content:', err)
          return false
        }
      }

      if (trySet()) return
      // retry a few times in case the editor mounts shortly after file selection
      let attempts = 0
      const maxAttempts = 10
      const interval = setInterval(() => {
        attempts += 1
        if (trySet() || attempts >= maxAttempts) {
          clearInterval(interval)
          if (attempts >= maxAttempts) console.warn('App: Giving up after retries to paste file into editor')
        }
      }, 200)
    }
    reader.onerror = (err) => {
      console.error('App: Error reading file:', err)
    }
    reader.readAsText(f)
  }

  const confirmarRmdisk = (entrada) => entrada;

  const analizar = async () => {
    const confirmar = window.confirm('¿Está seguro de ejecutar los comandos?');
    if (!confirmar) {
      return;
    }
    var entrada = editorRef.current.getValue();
    const entradaFiltrada = confirmarRmdisk(entrada);
    const response = await fetch('http://localhost:3000/mia', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ comando: entradaFiltrada }),
    });
    const data = await response.json();
    consolaRef.current.setValue(data.resultados.join('\n'));
  }

  return (
    <div className="min-h-screen flex flex-col">
      <Navbar
        onOpenLogin={() => { console.log('App: onOpenLogin called'); setShowLoginModal(true) }}
        onNavigate={(nv) => {
          console.log('App: navigate ->', nv)
          // change current view
          setView(nv)
          // clear selection when switching to disks/partitions
          if (nv === 'disks' || nv === 'partitions') setSelectedDisk(null)
        }}
        onExecute={analizar}
        onFileSelected={handleFileSelected}
        session={session}
        onLogout={async () => {
          console.log('App: logout clicked')
          try {
            await api.logout()
            // clear stored credentials
            localStorage.removeItem('mia_username')
            localStorage.removeItem('mia_password')
            localStorage.removeItem('mia_id')
            setSession(null)
            setToast('Sesión cerrada correctamente')
          } catch (err) {
            console.error('Logout failed', err)
            setToast('Error cerrando sesión')
            setSession(null)
          }
        }}
      />
      {/* Render the selected view panel as centered modals so the page doesn't shift */}
      {(view === 'disks' || view === 'partitions') && (
        <Modal title="Seleccionar disco" onClose={() => setView('editor')}>
          <DiskList onSelect={(diskPath, partitionName) => {
            console.log('App: Disk selected', diskPath, partitionName)
            setSelectedDisk({ diskPath, partitionName })
            setView('explorer')
          }} />
        </Modal>
      )}

      {view === 'explorer' && (
        {/* If user hasn't selected a disk/partition yet, show the host filesystem explorer (auto-populate to user's home).
            Otherwise show the explorer for the selected disk/partition. */}
        (!selectedDisk) ? (
          <Modal title="Explorador (Host)" onClose={() => setView('editor')}>
            <FileSystemViewer diskPath={"__hostfs"} partitionName={""} />
          </Modal>
        ) : (
          <Modal title={`Explorador: ${selectedDisk.partitionName || ''}`} onClose={() => setView('editor')}>
            <FileSystemViewer diskPath={selectedDisk.diskPath} partitionName={selectedDisk.partitionName} />
          </Modal>
        )
      )}

      {view === 'graph' && (
        <Modal title="Grafo del sistema" onClose={() => setView('editor')}>
          <GraphView />
        </Modal>
      )}
      {showLoginModal && (
        <Login
          onLogin={(s) => { setSession(s); console.log('App: setSession', s); setToast('Inicio de sesión exitoso') }}
          onClose={() => setShowLoginModal(false)}
          onLogout={() => { console.log('App: logout'); setSession(null) }}
          session={session}
        />
      )}

      {/* Toast / emergente simple */}
      {toast && (
        <div className="fixed top-4 right-4 z-50">
          <div className="bg-green-600 text-white px-4 py-2 rounded shadow">{toast}</div>
        </div>
      )}
  <main className="flex-1 flex flex-col py-3">
  <section className='w-full px-6 py-2'>
  <h1 className='mt-2 mb-4 text-2xl text-center'>Entrada</h1>
        <div className='flex justify-center w-full'>
          <div className="w-full max-w-full">
            <Editor className='rounded-md'
              height="28vh"
              width="100%"
              theme='vs-dark'
              defaultLanguage='cpp'
              defaultValue=''
              options={{
                scrollBeyondLastLine: false,
                fontSize: "16px"
              }}
              onMount={(editor) => handleEditor(editor, "editor")}
            />
          </div>
  </div>
      </section>
      <section className='w-full px-6 py-2'>
  <h1 className='mt-2 mb-4 text-2xl text-center'>Salida</h1>
        <div className='flex justify-center w-full'>
          <div className="w-full max-w-full">
          <Editor className='rounded-md'
            height="36vh"
            width="100%"
            theme='vs-dark'
            defaultLanguage='cpp'
            defaultValue=''
            options={{
              scrollBeyondLastLine: false,
              fontSize: "16px",
              readOnly: true
            }}
            onMount={(editor) => handleEditor(editor, "consola")}
          />
          </div>
        </div>
      </section>
  {/* file input moved into Navbar; no local input here anymore */}
      </main>
    </div>
  )
}

export default App
