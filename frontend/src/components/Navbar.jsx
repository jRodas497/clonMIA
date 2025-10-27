import React from 'react'

export default function Navbar({ onOpenLogin, onNavigate, onExecute, onFileSelected, session, onLogout }) {
  // local input ref to guarantee the file input is always present in the navbar
  const inputRef = React.useRef(null)

  const triggerFileInput = () => {
    console.log('Navbar: triggerFileInput')
    if (inputRef.current) inputRef.current.click()
    else console.warn('Navbar: inputRef not available')
  }

  const handleChange = (e) => {
    const f = e.target.files && e.target.files[0]
    if (!f) return
    if (typeof onFileSelected === 'function') onFileSelected(f)
    // clear the input so the same file can be selected again
    e.target.value = null
  }

  return (
    <header className="w-full bg-white shadow-md">
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
        <nav className="flex items-center gap-4">
          <button className="text-sm font-medium text-gray-700 hover:text-gray-900" onClick={() => onNavigate('disks')}>Selección de Disco</button>
          <button className="text-sm font-medium text-gray-700 hover:text-gray-900" onClick={() => onNavigate('partitions')}>Selección de Partición</button>
          <button className="text-sm font-medium text-gray-700 hover:text-gray-900" onClick={() => onNavigate('explorer')}>Navegación</button>
        </nav>

        <div className="flex items-center gap-3">
          {/* show current user (username(id)) when session exists; fall back to localStorage if needed */}
          {session && (
            <div className="mr-4 flex items-center gap-3">
              {/* Username (bold) and partition id as a small badge */}
              <div className="text-base font-semibold text-gray-800">
                {(() => {
                  try {
                    return session.username || (typeof window !== 'undefined' && window.localStorage && localStorage.getItem('mia_username')) || ''
                  } catch (e) {
                    return ''
                  }
                })()}
              </div>
              <div className="text-sm font-medium text-white bg-indigo-600 px-2 py-0.5 rounded">
                {(() => {
                  try {
                    return session.id || (typeof window !== 'undefined' && window.localStorage && localStorage.getItem('mia_id')) || ''
                  } catch (e) {
                    return ''
                  }
                })()}
              </div>
            </div>
          )}
          {onExecute && (
            <button type="button" onClick={onExecute} className="px-4 py-2 rounded-md bg-btn text-white hover:opacity-90">Ejecutar</button>
          )}
          <button type="button" onClick={triggerFileInput} className="px-4 py-2 rounded-md bg-green-600 text-white hover:bg-green-700">Subir archivo</button>
          {session ? (
            <button type="button" onClick={() => { console.log('Navbar: onLogout clicked'); if (typeof onLogout === 'function') onLogout(); else console.warn('onLogout not provided') }} className="px-4 py-2 rounded-md bg-red-600 text-white hover:bg-red-700">Cerrar sesión</button>
          ) : (
            <button type="button" onClick={() => { console.log('Navbar: onOpenLogin clicked'); if (typeof onOpenLogin === 'function') onOpenLogin(); else console.warn('onOpenLogin not provided') }} className="px-4 py-2 rounded-md bg-indigo-600 text-white hover:bg-indigo-700">Iniciar sesión</button>
          )}
        </div>
        {/* hidden file input inside navbar so it always exists when the navbar is mounted */}
        <input ref={inputRef} type="file" className="sr-only" onChange={handleChange} />
      </div>
    </header>
  )
}
