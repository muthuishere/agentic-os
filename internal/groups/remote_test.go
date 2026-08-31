package groups

import (
	"os"
	"path/filepath"
	"testing"
)

// noEnv is an environment that names nothing, so a test exercises the config
// file and the built-in defaults without the developer's own shell leaking in.
func noEnv(string) string { return "" }

func envWith(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func argsOf(t *testing.T, args ...string) *argSet {
	t.Helper()
	set, err := parseArgs(args, "monitor", "port", "fps", "key")
	if err != nil {
		t.Fatalf("parseArgs(%v): %v", args, err)
	}
	return set
}

func TestLoadRemoteConfigMissingFileIsNotAnError(t *testing.T) {
	cfg, err := loadRemoteConfig(filepath.Join(t.TempDir(), "remote.json"))
	if err != nil {
		t.Fatalf("a missing config is the normal case: %v", err)
	}
	if cfg.Key != nil || cfg.Port != nil || cfg.FPS != nil || cfg.Monitor != nil {
		t.Fatalf("expected an empty config, got %+v", cfg)
	}
}

// A malformed config must fail loudly: quietly falling back to a generated key
// would change who can reach the machine.
func TestLoadRemoteConfigRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRemoteConfig(path); err == nil {
		t.Fatal("expected an error for malformed remote.json")
	}
}

func TestLoadRemoteConfigReadsEveryField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote.json")
	body := `{"key":"config-key-long-enough","port":9123,"fps":4.5,"monitor":2}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRemoteConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Key == nil || *cfg.Key != "config-key-long-enough" {
		t.Fatalf("key: %v", cfg.Key)
	}
	if cfg.Port == nil || *cfg.Port != 9123 {
		t.Fatalf("port: %v", cfg.Port)
	}
	if cfg.FPS == nil || *cfg.FPS != 4.5 {
		t.Fatalf("fps: %v", cfg.FPS)
	}
	if cfg.Monitor == nil || *cfg.Monitor != 2 {
		t.Fatalf("monitor: %v", cfg.Monitor)
	}
}

func TestResolveRemoteOptionsBuiltInDefaults(t *testing.T) {
	opts, generated, err := resolveRemoteOptions(argsOf(t), remoteConfig{}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !generated {
		t.Fatal("with no key anywhere, the run must mint one and say so")
	}
	if opts.Key != "" {
		t.Fatalf("an unset key must stay empty so windowctl mints it, got %q", opts.Key)
	}
	if opts.FPS != remoteDefaultFPS || opts.Port != 0 || opts.Monitor != nil {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestResolveRemoteOptionsConfigBeatsDefaults(t *testing.T) {
	key := "config-key-long-enough"
	port, monitor := 9123, 2
	fps := 4.5
	opts, generated, err := resolveRemoteOptions(argsOf(t), remoteConfig{
		Key: &key, Port: &port, FPS: &fps, Monitor: &monitor,
	}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if generated {
		t.Fatal("a configured key is not a generated one")
	}
	if opts.Key != key || opts.Port != port || opts.FPS != fps || opts.Monitor == nil || *opts.Monitor != monitor {
		t.Fatalf("config was not applied: %+v", opts)
	}
}

func TestResolveRemoteOptionsFlagsBeatConfig(t *testing.T) {
	key := "config-key-long-enough"
	port, monitor := 9123, 2
	fps := 4.5
	cfg := remoteConfig{Key: &key, Port: &port, FPS: &fps, Monitor: &monitor}

	opts, _, err := resolveRemoteOptions(
		argsOf(t, "--key=flag-key-long-enough", "--port=7777", "--fps=8", "--monitor=3"),
		cfg, envWith(map[string]string{"AGENTIC_OS_REMOTE_KEY": "env-key-long-enough"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Key != "flag-key-long-enough" {
		t.Fatalf("--key must beat both the env var and the file, got %q", opts.Key)
	}
	if opts.Port != 7777 || opts.FPS != 8 || opts.Monitor == nil || *opts.Monitor != 3 {
		t.Fatalf("flags did not win: %+v", opts)
	}
}

// The env var is how a launcher injects a secret, so it beats the file on disk
// while still losing to a --key typed at the terminal.
func TestResolveRemoteOptionsEnvKeyBeatsConfig(t *testing.T) {
	key := "config-key-long-enough"
	opts, generated, err := resolveRemoteOptions(argsOf(t), remoteConfig{Key: &key},
		envWith(map[string]string{"AGENTIC_OS_REMOTE_KEY": "env-key-long-enough"}))
	if err != nil {
		t.Fatal(err)
	}
	if generated {
		t.Fatal("an env key is a configured key")
	}
	if opts.Key != "env-key-long-enough" {
		t.Fatalf("env var should win over the file, got %q", opts.Key)
	}
}

// windowctl's own env var must not be honoured here: the two tools are
// configured separately, so a key exported for the windowctl CLI cannot
// silently become the key for an aos share.
func TestResolveRemoteOptionsIgnoresWindowctlEnv(t *testing.T) {
	opts, generated, err := resolveRemoteOptions(argsOf(t), remoteConfig{},
		envWith(map[string]string{"WINDOWCTL_REMOTE_KEY": "windowctl-key-long-enough"}))
	if err != nil {
		t.Fatal(err)
	}
	if !generated || opts.Key != "" {
		t.Fatalf("WINDOWCTL_REMOTE_KEY must be ignored, got key %q", opts.Key)
	}
}

func TestResolveRemoteOptionsRejectsShortKey(t *testing.T) {
	for _, source := range []string{"flag", "env", "config"} {
		var (
			set = argsOf(t)
			cfg remoteConfig
			env = noEnv
		)
		switch source {
		case "flag":
			set = argsOf(t, "--key=short")
		case "env":
			env = envWith(map[string]string{"AGENTIC_OS_REMOTE_KEY": "short"})
		case "config":
			k := "short"
			cfg = remoteConfig{Key: &k}
		}
		if _, _, err := resolveRemoteOptions(set, cfg, env); err == nil {
			t.Fatalf("a 5-character key from the %s must be rejected", source)
		}
	}
}

func TestResolveRemoteOptionsRejectsBadValues(t *testing.T) {
	cases := map[string][]string{
		"monitor below 1":   {"--monitor=0"},
		"non-numeric fps":   {"--fps=fast"},
		"zero fps":          {"--fps=0"},
		"port out of range": {"--port=70000"},
	}
	for name, args := range cases {
		if _, _, err := resolveRemoteOptions(argsOf(t, args...), remoteConfig{}, noEnv); err == nil {
			t.Fatalf("%s should have been rejected", name)
		}
	}
}

// A recorded share whose port no longer answers is history, not a live link.
func TestReadRemoteStateRejectsDeadShare(t *testing.T) {
	dir := t.TempDir()
	env := envWith(map[string]string{"AGENTIC_OS_CONFIG_DIR": dir})
	if err := writeRemoteState(env, remoteState{URL: "http://x/?t=k", Port: 1, PID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, live := readRemoteState(env); live {
		t.Fatal("a share on a port nothing is listening on must not read as live")
	}
	clearRemoteState(env)
	if _, live := readRemoteState(env); live {
		t.Fatal("a cleared share must not read as live")
	}
}
