package agentipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// maxEventBytes caps the encoded event, trailing newline included, because
// the Emacs receiver rejects event files larger than 1 MiB (spec.md § 2.3).
const maxEventBytes = 1 << 20

// defaultTimeout bounds one emacsclient call when neither --timeout nor
// IMOOGI_AGENT_TIMEOUT is set (plan.md D-9).
const defaultTimeout = 5 * time.Second

// timeoutEnv overrides the default time limit; --timeout overrides it.
const timeoutEnv = "IMOOGI_AGENT_TIMEOUT"

// usageError marks an invocation rejected before anything is sent (exit 2).
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// request is one parsed invocation: the encoded event and its time limit.
type request struct {
	event   []byte
	timeout time.Duration
}

// envelope is the protocol version 1 wrapper (spec.md § 2.1). The field order
// here is the order on the wire; optional fields are pointers so an option
// given with an empty value is still sent, while an absent one is omitted.
type envelope struct {
	Version   string  `json:"version"`
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	Project   *string `json:"project,omitempty"`
	Session   *string `json:"session,omitempty"`
	Payload   any     `json:"payload"`
}

type messagePayload struct {
	Text string `json:"text"`
}

type pathPayload struct {
	Path string `json:"path"`
}

type gotoPayload struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column *int   `json:"column,omitempty"`
}

type artifactPayload struct {
	Path         string  `json:"path"`
	ArtifactType *string `json:"artifactType,omitempty"`
	Title        *string `json:"title,omitempty"`
}

type finishPayload struct {
	Status   string  `json:"status"`
	Summary  *string `json:"summary,omitempty"`
	Artifact *string `json:"artifact,omitempty"`
}

// commonOptions may appear with every subcommand.
var commonOptions = []string{"--project", "--session", "--timeout"}

// subcommandOptions lists the options each subcommand accepts beyond the
// common ones.
var subcommandOptions = map[string][]string{
	"message":   nil,
	"open-file": nil,
	"goto":      nil,
	"artifact":  {"--type", "--title"},
	"finish":    {"--artifact"},
}

// parseArgs splits args into positional arguments and option values. Options
// may appear anywhere; everything after "--" is positional. Each option takes
// exactly one value, the next argument, taken literally.
func parseArgs(args []string) (positional []string, options map[string]string, err error) {
	options = map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		if !isKnownOption(arg) {
			return nil, nil, usagef("unknown option %s", arg)
		}
		if i+1 >= len(args) {
			return nil, nil, usagef("option %s needs a value", arg)
		}
		if _, dup := options[arg]; dup {
			return nil, nil, usagef("option %s given more than once", arg)
		}
		options[arg] = args[i+1]
		i++
	}
	return positional, options, nil
}

func isKnownOption(name string) bool {
	if slices.Contains(commonOptions, name) {
		return true
	}
	for _, opts := range subcommandOptions {
		if slices.Contains(opts, name) {
			return true
		}
	}
	return false
}

// parseRequest turns an invocation into an encoded event. getwd is consulted
// only when a relative path needs absolutizing; a failure there is an
// environment problem, not a usage error.
func parseRequest(args []string, getenv func(string) string, getwd func() (string, error), now time.Time) (*request, error) {
	positional, options, err := parseArgs(args)
	if err != nil {
		return nil, err
	}
	if len(positional) == 0 {
		return nil, usagef("missing subcommand (message, open-file, goto, artifact, finish)")
	}
	sub, rest := positional[0], positional[1:]
	allowed, ok := subcommandOptions[sub]
	if !ok {
		return nil, usagef("unknown subcommand %q", sub)
	}
	for name := range options {
		if !slices.Contains(commonOptions, name) && !slices.Contains(allowed, name) {
			return nil, usagef("option %s is not valid for %s", name, sub)
		}
	}

	abs := func(path string) (string, error) {
		if path == "" {
			return "", usagef("empty path")
		}
		if filepath.IsAbs(path) {
			return path, nil
		}
		dir, err := getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine working directory: %w", err)
		}
		return filepath.Join(dir, path), nil
	}

	event := envelope{Version: "1", Timestamp: now.Format(time.RFC3339)}
	event.Project = optional(options, "--project")
	event.Session = optional(options, "--session")

	switch sub {
	case "message":
		if len(rest) != 1 {
			return nil, usagef("message takes exactly one TEXT argument")
		}
		if rest[0] == "" {
			return nil, usagef("message TEXT must not be empty")
		}
		event.Type = "message"
		event.Payload = messagePayload{Text: rest[0]}
	case "open-file":
		if len(rest) != 1 {
			return nil, usagef("open-file takes exactly one PATH argument")
		}
		path, err := abs(rest[0])
		if err != nil {
			return nil, err
		}
		event.Type = "open-file"
		event.Payload = pathPayload{Path: path}
	case "goto":
		if len(rest) < 2 || len(rest) > 3 {
			return nil, usagef("goto takes PATH LINE [COL]")
		}
		line, err := positiveInt("LINE", rest[1])
		if err != nil {
			return nil, err
		}
		payload := gotoPayload{Line: line}
		if len(rest) == 3 {
			col, err := positiveInt("COL", rest[2])
			if err != nil {
				return nil, err
			}
			payload.Column = &col
		}
		if payload.Path, err = abs(rest[0]); err != nil {
			return nil, err
		}
		event.Type = "goto-location"
		event.Payload = payload
	case "artifact":
		if len(rest) != 1 {
			return nil, usagef("artifact takes exactly one PATH argument")
		}
		path, err := abs(rest[0])
		if err != nil {
			return nil, err
		}
		event.Type = "artifact-created"
		event.Payload = artifactPayload{
			Path:         path,
			ArtifactType: optional(options, "--type"),
			Title:        optional(options, "--title"),
		}
	case "finish":
		if len(rest) < 1 || len(rest) > 2 {
			return nil, usagef("finish takes success|failed [SUMMARY]")
		}
		if rest[0] != "success" && rest[0] != "failed" {
			return nil, usagef("finish status must be success or failed, not %q", rest[0])
		}
		payload := finishPayload{Status: rest[0]}
		if len(rest) == 2 {
			payload.Summary = &rest[1]
		}
		if artifact := optional(options, "--artifact"); artifact != nil {
			path, err := abs(*artifact)
			if err != nil {
				return nil, err
			}
			payload.Artifact = &path
		}
		event.Type = "task-finished"
		event.Payload = payload
	}

	timeout, err := resolveTimeout(options, getenv)
	if err != nil {
		return nil, err
	}
	data, err := encodeEvent(event)
	if err != nil {
		return nil, err
	}
	return &request{event: data, timeout: timeout}, nil
}

// encodeEvent serializes without HTML escaping and enforces the size cap on
// the exact bytes that will be written to the event file.
func encodeEvent(event envelope) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(event); err != nil {
		return nil, fmt.Errorf("cannot encode event: %w", err)
	}
	if buf.Len() > maxEventBytes {
		return nil, usagef("event is %d bytes, larger than the 1 MiB limit", buf.Len())
	}
	return buf.Bytes(), nil
}

// resolveTimeout applies --timeout > IMOOGI_AGENT_TIMEOUT > 5s. The
// environment variable is read only when the option is absent.
func resolveTimeout(options map[string]string, getenv func(string) string) (time.Duration, error) {
	if value, ok := options["--timeout"]; ok {
		return positiveSeconds("--timeout", value)
	}
	if value := getenv(timeoutEnv); value != "" {
		return positiveSeconds(timeoutEnv, value)
	}
	return defaultTimeout, nil
}

func positiveSeconds(name, value string) (time.Duration, error) {
	n, err := positiveInt(name, value)
	if err != nil {
		return 0, err
	}
	if int64(n) > math.MaxInt64/int64(time.Second) {
		return 0, usagef("%s %s is too large", name, value)
	}
	return time.Duration(n) * time.Second, nil
}

// positiveInt accepts only decimal digits (no sign) denoting a value >= 1.
func positiveInt(name, value string) (int, error) {
	if value == "" || strings.TrimLeft(value, "0123456789") != "" {
		return 0, usagef("%s must be a positive integer, not %q", name, value)
	}
	n, err := strconv.Atoi(value)
	if errors.Is(err, strconv.ErrRange) {
		return 0, usagef("%s %s is too large", name, value)
	}
	if err != nil || n < 1 {
		return 0, usagef("%s must be a positive integer, not %q", name, value)
	}
	return n, nil
}

func optional(options map[string]string, name string) *string {
	value, ok := options[name]
	if !ok {
		return nil
	}
	return &value
}
