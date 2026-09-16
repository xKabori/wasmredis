import { observer, useSelector } from '@legendapp/state/react'
import { values$ } from './store'
import { ROW_HEIGHT } from './constants'

type Props = {
  entryKey: string
  top: number
}

export const Row = observer(function Row({ entryKey, top }: Props) {
  const value = useSelector(values$[entryKey])

  return (
    <div
      style={{
        position: 'absolute',
        top,
        height: ROW_HEIGHT,
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        width: '100%',
        boxSizing: 'border-box',
        borderBottom: '1px solid #cd2020',
      }}
    >
      <span style={{ width: 200 }}>{entryKey}</span>
      <span>{value}</span>
    </div>
  )
})
