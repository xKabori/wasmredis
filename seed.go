package wasmredis

import "strconv"

func (e *Engine) Seed(n int) {
	for i := 0; i < n; i++ {
		suffix := strconv.Itoa(i)
		e.Set("key:"+suffix, "value:"+suffix)
	}
}
