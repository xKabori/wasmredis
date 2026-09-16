import { observable } from '@legendapp/state'
import { db, loadWasm, seed } from './db'

export const keys$ = observable<string[]>([])

export const values$ = observable<Record<string, string>>({})

export async function load() {
  await loadWasm()

  if ((await db.getAll()).length === 0) {
    seed(100000)
  }

  const entries = await db.getAll()

  const keys: string[] = []
  const values: Record<string, string> = {}
  for (const entry of entries) {
    keys.push(entry.key)
    values[entry.key] = entry.value
  }

  keys$.set(keys)
  values$.set(values)
}
