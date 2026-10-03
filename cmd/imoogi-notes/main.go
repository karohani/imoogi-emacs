package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/karohani/imoogi-emacs/internal/notecache"
	"github.com/karohani/imoogi-emacs/internal/notemove"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "imoogi-notes %s\n", version)
		return 0
	}
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: imoogi-notes [--version]")
		return 2
	}
	data, err := readRequest(stdin)
	if err != nil {
		write(stdout, stderr, notemove.Response{OK: false, Code: "invalid_request", Error: err.Error()})
		return 0
	}
	var envelope struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		write(stdout, stderr, notemove.Response{OK: false, Code: "invalid_request", Error: err.Error()})
		return 0
	}
	if envelope.Operation == "move" {
		var request notemove.Request
		if err := json.Unmarshal(data, &request); err != nil {
			write(stdout, stderr, notemove.Response{OK: false, Code: "invalid_request", Error: err.Error()})
			return 0
		}
		write(stdout, stderr, notemove.Move(request))
		return 0
	}
	var request notecache.Request
	if err := json.Unmarshal(data, &request); err != nil {
		write(stdout, stderr, notecache.Response{Version: notecache.ProtocolVersion, OK: false, Status: "error", Errors: []notecache.Error{{Code: "invalid_request", Message: err.Error()}}})
		return 0
	}
	write(stdout, stderr, notecache.Run(context.Background(), request))
	return 0
}

func readRequest(reader io.Reader) ([]byte, error) {
	const limit = 16 << 20
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("request exceeds %d bytes", limit)
	}
	return data, nil
}

func write(stdout, stderr io.Writer, response any) {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintln(stderr, err)
	}
}
