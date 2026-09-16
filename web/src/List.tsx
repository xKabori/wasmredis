import { useState } from 'react'
import { observer, useSelector } from '@legendapp/state/react'
import { keys$ } from './store'
import { Row } from './Row'
import { ROW_HEIGHT, VIEWPORT_HEIGHT, OVERSCAN } from './constants'

export const List = observer(function List() {
  const total = useSelector(() => keys$.length)
  const [scrollTop, setScrollTop] = useState(0)

  const firstVisible = Math.floor(scrollTop / ROW_HEIGHT)
  const visibleCount = Math.ceil(VIEWPORT_HEIGHT / ROW_HEIGHT)

  const startIndex = Math.max(0, firstVisible - OVERSCAN)
  const endIndex = Math.min(total, firstVisible + visibleCount + OVERSCAN)

  const rows = []
  for (let i = startIndex; i < endIndex; i++) {
    const key = keys$[i].peek()
    rows.push(<Row key={key} entryKey={key} top={i * ROW_HEIGHT} />)
  }

  return (
    <div
      onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
      style={{
        height: VIEWPORT_HEIGHT,
        overflowY: 'auto',
        position: 'relative',
        border: '1px solid #999',
      }}
    >
      <div style={{ height: total * ROW_HEIGHT, position: 'relative' }}>
        {rows}
      </div>
    </div>
  )
})
