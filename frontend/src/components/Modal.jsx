import React from 'react'

export default function Modal({ title, children, onClose }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50" onClick={(e) => { if (e.target === e.currentTarget && typeof onClose === 'function') onClose() }}>
      <div className="bg-white rounded-lg shadow-lg w-[70vw] h-[65vh] max-w-[1100px] max-h-[80vh] overflow-auto" role="dialog" aria-modal="true">
        <div className="flex items-center justify-between p-4 border-b">
          <div className="text-lg font-semibold">{title}</div>
          <div>
            <button onClick={() => { if (typeof onClose === 'function') onClose() }} className="px-3 py-1 bg-gray-200 rounded hover:bg-gray-300">Cerrar</button>
          </div>
        </div>
        <div className="p-4 h-full overflow-auto">
          {children}
        </div>
      </div>
    </div>
  )
}
