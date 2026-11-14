package doctor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/karohani/imoogi-emacs/internal/config"
	"github.com/karohani/imoogi-emacs/internal/lang/acp"
)

const fixToolchain = "make toolchain-setup"

// installRecord is the part of setup's install.json the doctor compares.
type installRecord struct {
	Bundle     string `json:"bundle"`
	LockSHA256 string `json:"lock_sha256"`
	Components []struct {
		Name           string `json:"name"`
		ArtifactSHA256 string `json:"artifact_sha256"`
	} `json:"components"`
	Skipped []string `json:"skipped"` // local-build components setup could not build
}

// checkAdapters reports, for every ACP adapter in toolchains.lock.json, whether
// the active toolchain bundle installed it from this lock and it answers an ACP
// initialize request, and whether the agent CLI it drives can be found.
func checkAdapters(env Env) []Finding {
	lockData, err := env.ReadFile(filepath.Join(env.Root(), "toolchains.lock.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var lock config.ResolvedLock
	if err == nil {
		err = json.Unmarshal(lockData, &lock)
	}
	if err != nil {
		return []Finding{{Kind: "acp", Name: "toolchains.lock.json", Status: StatusUnknown, Detail: err.Error()}}
	}
	var adapters []config.LockComponent
	for _, c := range lock.Components {
		if c.Kind == config.KindACPAdapter {
			adapters = append(adapters, c)
		}
	}
	if len(adapters) == 0 {
		return nil
	}
	sum := sha256.Sum256(lockData)
	state := bundleState(env, lock.Bundle, hex.EncodeToString(sum[:]))
	var out []Finding
	for _, c := range adapters {
		out = append(out, checkAdapter(env, c, state))
		if a, ok := acp.Lookup(c.Name); ok {
			out = append(out, checkCLI(env, a))
		}
	}
	return out
}

// bundleState explains why the active toolchain is not the one this lock
// describes ("" when it is), with the record setup wrote when it installed it.
type installState struct {
	status, detail string
	record         installRecord
}

func bundleState(env Env, bundle, lockSum string) installState {
	data, err := env.ReadFile(filepath.Join(env.Root(), ".local", "toolchains", bundle, "install.json"))
	if err != nil {
		return installState{status: StatusMissing, detail: "toolchain bundle " + bundle + " is not installed"}
	}
	var record installRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return installState{status: StatusUnknown, detail: "install.json: " + err.Error()}
	}
	if record.LockSHA256 != lockSum {
		return installState{status: StatusStale, detail: "bundle " + bundle + " was installed from a different toolchains.lock.json"}
	}
	want := filepath.Join("toolchains", bundle, "bin")
	if target, err := env.Readlink(filepath.Join(env.Root(), ".local", "bin")); err != nil || filepath.Clean(target) != want {
		if err != nil {
			target = "nothing"
		}
		return installState{status: StatusStale, detail: ".local/bin points to " + target + ", not " + want}
	}
	return installState{record: record}
}

func checkAdapter(env Env, c config.LockComponent, state installState) Finding {
	f := Finding{Kind: "acp", Name: c.Name, Status: StatusOK, Fix: fixToolchain}
	if state.status != "" {
		f.Status, f.Detail = state.status, state.detail
		return f
	}
	for _, name := range state.record.Skipped {
		if name == c.Name {
			f.Status = StatusMissing
			f.Detail = "skipped by setup: its locally built bundle could not be built (npm registry unreachable?)"
			f.Fix = fixToolchain + " once online, or copy a built bundle to " + c.Artifact.Path + " and run " + fixToolchain
			return f
		}
	}
	installed := ""
	for _, rc := range state.record.Components {
		if rc.Name == c.Name {
			installed = rc.ArtifactSHA256
		}
	}
	if installed != c.Artifact.SHA256 {
		f.Status, f.Detail = StatusStale, "installed artifact differs from the locked "+c.Name+" "+c.UpstreamVersion
		return f
	}
	launcher := filepath.Join(env.Root(), ".local", "bin", c.Name)
	if !env.Exists(launcher) {
		f.Status, f.Detail = StatusMissing, "launcher .local/bin/"+c.Name+" is missing"
		return f
	}
	if err := env.Handshake(launcher); err != nil {
		f.Status, f.Detail = StatusFailed, "ACP initialize failed: "+err.Error()
		return f
	}
	f.Fix, f.Detail = "", c.UpstreamVersion+" answers ACP initialize (bundle "+state.record.Bundle+")"
	return f
}

func checkCLI(env Env, a acp.Adapter) Finding {
	f := Finding{Kind: "cli", Name: a.CLI, Status: StatusOK}
	fix := "install the " + a.CLI + " CLI on PATH or set " + a.EnvVar
	if v := env.Getenv(a.EnvVar); v != "" {
		if env.Exists(v) {
			f.Detail = a.EnvVar + "=" + v
			return f
		}
		f.Status, f.Detail, f.Fix = StatusWarn, a.EnvVar+"="+v+" does not exist; "+a.Name+" cannot start sessions", fix
		return f
	}
	path, err := env.LookPath(a.CLI)
	if err != nil {
		f.Status, f.Detail, f.Fix = StatusWarn, a.CLI+" is not on PATH; "+a.Name+" answers but cannot start sessions", fix
		return f
	}
	f.Detail = "→ " + path + " (used by " + a.Name + ")"
	return f
}
