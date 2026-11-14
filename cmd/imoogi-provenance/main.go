package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/karohani/imoogi-emacs/internal/provenance"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "outdated" {
		outdated(os.Args[2:])
		return
	}
	valid := len(os.Args) == 2 && (os.Args[1] == "verify" || os.Args[1] == "generate")
	valid = valid || (len(os.Args) == 5 && os.Args[1] == "record-git-source")
	if !valid {
		fmt.Fprintln(os.Stderr, "usage: imoogi-provenance <verify|generate|outdated [--json] [--all]|record-git-source ID COMMIT REF>")
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

var statusOrder = map[string]int{
	provenance.StatusUpdate: 0, provenance.StatusMoved: 1, provenance.StatusWatch: 2,
	provenance.StatusUnknown: 3, provenance.StatusCurrent: 4, provenance.StatusSkipped: 5,
}

// outdated reports bundled components with newer upstream versions. It only reads.
func outdated(args []string) {
	fs := flag.NewFlagSet("outdated", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print results as JSON")
	all := fs.Bool("all", false, "include components that are up to date or skipped")
	_ = fs.Parse(args)
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	results, err := provenance.Outdated(context.Background(), root, "vendor-manifest.json",
		provenance.NewNetRemote(), provenance.LocalTools{}, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if statusOrder[a.Status] != statusOrder[b.Status] {
			return statusOrder[a.Status] < statusOrder[b.Status]
		}
		if a.Domain != b.Domain {
			return a.Domain < b.Domain
		}
		return a.ID < b.ID
	})
	counts := map[string]int{}
	shown := results[:0:0]
	for _, r := range results {
		counts[r.Status]++
		if *all || (r.Status != provenance.StatusCurrent && r.Status != provenance.StatusSkipped) {
			shown = append(shown, r)
		}
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(shown)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tDOMAIN\tID\tCURRENT\tLATEST\tLEVEL\tDETAIL")
	for _, r := range shown {
		detail := strings.ReplaceAll(r.Detail, "\n", " | ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.Status, r.Domain, r.ID, r.Current, r.Latest, r.Level, detail)
	}
	_ = w.Flush()
	fmt.Printf("\n%d checked: %d update, %d moved, %d watch, %d unknown, %d current, %d skipped\n", len(results),
		counts[provenance.StatusUpdate], counts[provenance.StatusMoved], counts[provenance.StatusWatch],
		counts[provenance.StatusUnknown], counts[provenance.StatusCurrent], counts[provenance.StatusSkipped])
}
