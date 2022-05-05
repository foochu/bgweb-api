package main

import (
	"bgweb-api/internal/api"
	"bgweb-api/internal/gnubg"
	"bgweb-api/internal/openapi"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"syscall/js"
)

//go:embed data
var data embed.FS

func main() {
	c := make(chan struct{}, 0)

	// root embedded fs to data/
	dataDir, err := fs.Sub(data, "data")
	if err != nil {
		panic(err)
	}

	if err := gnubg.Init(dataDir); err != nil {
		panic(err)
	}

	println("WASM Go Initialized")
	// register functions
	{
		js.Global().Set("wasm_get_moves", js.FuncOf(getMoves))
	}
	<-c
}

func getMoves(this js.Value, input []js.Value) interface{} {
	var args openapi.MoveArgs

	if err := json.Unmarshal([]byte(input[0].String()), &args); err != nil {
		return fmt.Sprintf("{\"error\": \"%v\"}", err.Error())
	}

	moves, err := api.GetMoves(args)

	if err != nil {
		return fmt.Sprintf("{\"error\": \"%v\"}", err.Error())
	}

	bytes, err := json.Marshal(moves)

	if err != nil {
		return fmt.Sprintf("{\"error\": \"%v\"}", err.Error())
	}

	return js.ValueOf(string(bytes))
}
