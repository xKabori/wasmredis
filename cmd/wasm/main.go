//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"wasmredis"
)

var engine *wasmredis.Engine

func main() {
	engine = wasmredis.NewEngine(wasmredis.NewMemoryStorage())

	engine.Start()

	js.Global().Set("wasmGetAll", js.FuncOf(getAll))
	js.Global().Set("wasmSeed", js.FuncOf(seed))

	select {}
}

func getAll(this js.Value, args []js.Value) any {
	data, err := json.Marshal(engine.GetAll())
	if err != nil {
		return ""
	}
	return string(data)
}

func seed(this js.Value, args []js.Value) any {
	engine.Seed(args[0].Int())
	return nil
}
