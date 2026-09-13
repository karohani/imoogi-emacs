package main

import (
	"fmt"
	"os"

	"github.com/karohani/imoogi-emacs/internal/provenance"
)

func main() {
	valid := len(os.Args) == 2 && (os.Args[1] == "verify" || os.Args[1] == "generate")
	valid = valid || (len(os.Args) == 5 && os.Args[1] == "record-git-source")
	if !valid {
		fmt.Fprintln(os.Stderr, "usage: imoogi-provenance <verify|generate|record-git-source ID COMMIT REF>")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if os.Args[1] == "record-git-source" {
		if err := provenance.RecordGitSource(root, "provenance/sources.json", os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if os.Args[1] == "generate" {
		if err := provenance.Generate(root, "provenance/sources.json", "vendor-manifest.json"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	issues := provenance.Verify(root, "vendor-manifest.json")
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "vendor provenance: %v\n", issue)
		}
		os.Exit(1)
	}
	fmt.Println("vendor provenance: verified")
}
