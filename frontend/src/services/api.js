// Pequeño wrapper para llamadas a los endpoints del backend
const API_BASE = '' // usa el mismo host/origen desde el front (asume proxy o mismo dominio)

async function postJson(path, body) {
  const res = await fetch(API_BASE + path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`)
  return res.json()
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
