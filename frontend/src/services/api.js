// Pequeño wrapper para llamadas a los endpoints del backend
// Usar la URL completa del backend para evitar problemas de origen en desarrollo.
// En producción puedes dejarlo vacío para usar el mismo origen o configurar una variable de entorno.
let API_BASE = ''
if (typeof window !== 'undefined') {
  const h = window.location.hostname || ''
  // In dev we commonly run frontend on localhost or 127.0.0.1 — point API to backend
  if (h === 'localhost' || h === '127.0.0.1') {
    API_BASE = 'http://localhost:3000'
  } else {
    API_BASE = ''
  }
}

async function postJson(path, body) {
  let res
  try {
    res = await fetch(API_BASE + path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  } catch (err) {
    // network error or CORS blocked
    throw new Error('Network error: ' + (err.message || err))
  }

  const text = await res.text()
  // try parse JSON if any
  let data
  try {
    data = text ? JSON.parse(text) : null
  } catch (e) {
    // not JSON
    data = text
  }

  if (!res.ok) {
    const msg = (data && data.message) || res.statusText || 'HTTP error'
    throw new Error(msg)
  }

  return data
}

export async function login(username, password, id) {
  return postJson('/users/login', { username, password, id })
}

export async function listPartitions(diskPath) {
  return postJson('/api/disk/partitions', { path: diskPath })
}

export async function getPartitionTree(diskPath, partitionName) {
  return postJson('/api/disk/partition/tree', { diskPath, partitionName })
}

export async function getGraphDot() {
  const res = await fetch('/disk/partition/grafico')
  if (!res.ok) throw new Error('Failed to fetch graph')
  return res.text()
}

export async function listPath(diskPath, partitionName, path) {
  return postJson('/api/disk/partition/list', { diskPath, partitionName, path })
}

export async function statPath(diskPath, partitionName, path) {
  return postJson('/api/disk/partition/stat', { diskPath, partitionName, path })
}

export async function logout() {
  // Logout endpoint expects no body
  const res = await fetch(API_BASE + '/users/logout', { method: 'POST' })
  if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`)
  return res.json()
}

// Fallback: leer un archivo usando el analizador (comando cat) si no existe endpoint directo
export async function readFileByCat(path) {
  const cmd = `cat -path=${path}`
  const res = await postJson('/analyze', { command: cmd })
  // /analyze in 00106 returns { results: [...] } where output lines are in results
  if (res && res.results) return res.results.join('\n')
  return ''
}

export default { login, listPartitions, getPartitionTree, getGraphDot, readFileByCat, listPath, statPath }
