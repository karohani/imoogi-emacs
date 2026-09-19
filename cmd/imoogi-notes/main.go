package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/karohani/imoogi-emacs/internal/notemove"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("imoogi-notes %s\n", version)
		return
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: imoogi-notes [--version]")
		os.Exit(2)
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil {
		write(notemove.Response{OK: false, Code: "invalid_request", Error: err.Error()})
		return
	}
	var request notemove.Request
	if err := json.Unmarshal(data, &request); err != nil {
		write(notemove.Response{OK: false, Code: "invalid_request", Error: err.Error()})
		return
	}
	write(notemove.Move(request))
}

func write(response notemove.Response) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
