package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/karohani/imoogi-emacs/internal/clipboard"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("imoogi-clip %s\n", version)
		return
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: imoogi-clip [--version]")
		os.Exit(2)
	}
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil {
		write(clipboard.Failure("", "", clipboard.CodeInvalidRequest, err.Error()))
		return
	}
	request, err := clipboard.DecodeRequest(data)
	if err != nil {
		write(clipboard.Failure("", "", clipboard.CodeInvalidRequest, err.Error()))
		return
	}
	response := (clipboard.Service{Version: version}).Handle(context.Background(), request)
	write(response)
}

func write(response clipboard.Response) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
