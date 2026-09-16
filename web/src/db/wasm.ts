import type { Db, Entry } from './types'

declare global {
  class Go {
    importObject: WebAssembly.Imports
    run(instance: WebAssembly.Instance): void
  }
  function wasmGetAll(): string
  function wasmSeed(n: number): void
}

export async function loadWasm() {
  await new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    script.src = '/wasm_exec.js'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('wasm_exec.js introuvable'))
    document.head.appendChild(script)
  })

  const go = new Go()
  const result = await WebAssembly.instantiateStreaming(
    fetch('/main.wasm'),
    go.importObject,
  )

  go.run(result.instance)
}

export function createWasmDb(): Db {
  return {
    async getAll(): Promise<Entry[]> {
      return JSON.parse(wasmGetAll())
    },
  }
}

export function seed(n: number) {
  wasmSeed(n)
}
