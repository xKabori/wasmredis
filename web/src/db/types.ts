export type Entry = {
  key: string
  value: string
}

export type Db = {
  getAll(): Promise<Entry[]>
}
