// Command imoogi-doctor checks that bin/ tools and the installer's links come
// from the current checkout. Run it with `make doctor` (go run), so the doctor
// itself is always current. --fix rebuilds and relinks what is out of date.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/karohani/imoogi-emacs/internal/doctor"
)

func main() {
	fix := flag.Bool("fix", false, "rebuild stale tools, remove obsolete ones and relink")
	asJSON := flag.Bool("json", false, "print findings as JSON")
	flag.Parse()
	cwd, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	env, err := doctor.NewSystemEnv(cwd)
	if err != nil {
		fail(err)
	}
	findings := doctor.Check(env)
	if *fix && !doctor.Healthy(findings) {
		repair(env, findings)
		findings = doctor.Check(env)
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(findings)
	} else {
		print(findings)
	}
	if !doctor.Healthy(findings) {
		os.Exit(1)
	}
}

// repair runs the repository's own build and install steps; it never edits files itself
// except removing binaries this module built for tools that no longer exist.
func repair(env doctor.SystemEnv, findings []doctor.Finding) {
	build, install := false, false
	for _, f := range findings {
		switch {
		case f.Kind == "tool" && f.Status != doctor.StatusOK:
			build = true
		case f.Kind == "bin" && f.Status == doctor.StatusObsolete:
			path := filepath.Join(env.Root(), f.Name)
			fmt.Println("== remove", path)
			if err := os.Remove(path); err != nil {
				fmt.Fprintln(os.Stderr, "   ", err)
			}
		case f.Kind == "link" && f.Status != doctor.StatusOK:
			install = true
		}
	}
	if build {
		step(env.Root(), "make", "build-all")
	}
	if install {
		step(env.Root(), "./scripts/install.sh", "--without-toolchain")
	}
}

func step(dir, name string, args ...string) {
	fmt.Printf("== %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "   %s failed: %v\n", name, err)
	}
}

func print(findings []doctor.Finding) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tNAME\tDETAIL\tFIX")
	bad := 0
	for _, f := range findings {
		if f.Status != doctor.StatusOK {
			bad++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", f.Status, f.Name, f.Detail, f.Fix)
	}
	_ = w.Flush()
	if doctor.Healthy(findings) {
		fmt.Printf("\nimoogi doctor: all %d checks ok", len(findings))
		if bad > 0 {
			fmt.Printf(" (%d warning)", bad)
		}
		fmt.Println()
		return
	}
	fmt.Printf("\nimoogi doctor: %d of %d checks need attention; run `make doctor ARGS=--fix`\n", bad, len(findings))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "imoogi doctor:", err)
	os.Exit(2)
}
