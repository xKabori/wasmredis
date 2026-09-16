import { useEffect, useState } from 'react'
import { load } from './store'
import { List } from './List'

export default function App() {
  const [ready, setReady] = useState(false)

  useEffect(() => {
    load().then(() => setReady(true))
  }, [])

  if (!ready) {
    return <div>Chargement</div>
  }

  return <List />
}
