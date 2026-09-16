import { createWasmDb } from './wasm'

export type { Db, Entry } from './types'
export { loadWasm, seed } from './wasm'

export const db = createWasmDb()
