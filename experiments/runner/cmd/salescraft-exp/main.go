package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TrialConfig struct {
	TrialID         string            `json:"trial_id"`
	Phase           string            `json:"phase"`
	AllowedVariable string            `json:"allowed_variable"`
	BaselineRef     string            `json:"baseline_ref"`
	RepoRoot        string            `json:"repo_root"`
	WorkspaceRoot   string            `json:"workspace_root"`
	ArtifactRoot    string            `json:"artifact_root"`
	CachePolicy     string            `json:"cache_policy"`
	Tool            ToolConfig        `json:"tool"`
	Models          map[string]string `json:"models"`
	Sampling        map[string]any    `json:"sampling"`
	Loop            map[string]any    `json:"loop"`
	Context         map[string]any    `json:"context"`
	Capabilities    map[string]any    `json:"capabilities"`
	Verification    Verification      `json:"verification"`

	configPath string
}

type ToolConfig struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type Verification struct {
	Commands [][]string `json:"commands"`
}

type CommandResult struct {
	Command    []string  `json:"command"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	DurationMS int64     `json:"duration_ms"`
	ExitCode   int       `json:"exit_code"`
	Error      string    `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usageAndExit()
	}

	cmd := os.Args[1]
	configPath := parseConfigFlag(os.Args[2:])
	cfg, err := loadConfig(configPath)
	exitOnErr(err)

	switch cmd {
	case "prepare":
		err = prepare(cfg)
	case "run":
		err = runTool(cfg)
	case "verify":
		err = verify(cfg)
	case "archive":
		err = archive(cfg)
	case "clean":
		err = clean(cfg)
	case "trial":
		err = trial(cfg)
	default:
		usageAndExit()
	}
	exitOnErr(err)
}

func usageAndExit() {
	fmt.Fprintf(os.Stderr, "usage: salescraft-exp {prepare|run|verify|archive|clean|trial} --config path/to/trial.json\n")
	os.Exit(2)
}

func parseConfigFlag(args []string) string {
	fs := flag.NewFlagSet("salescraft-exp", flag.ExitOnError)
	configPath := fs.String("config", "", "trial config JSON path")
	_ = fs.Parse(args)
	if *configPath == "" {
		usageAndExit()
	}
	return *configPath
}

func loadConfig(path string) (TrialConfig, error) {
	var cfg TrialConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.configPath = path

	if cfg.TrialID == "" {
		return cfg, errors.New("trial_id is required")
	}
	if cfg.BaselineRef == "" {
		return cfg, errors.New("baseline_ref is required")
	}
	if cfg.RepoRoot == "" {
		cfg.RepoRoot = "."
	}
	if cfg.WorkspaceRoot == "" {
		cfg.WorkspaceRoot = "experiments/trials"
	}
	if cfg.ArtifactRoot == "" {
		cfg.ArtifactRoot = "experiments/artifacts"
	}
	if cfg.Tool.Command == "" {
		return cfg, errors.New("tool.command is required")
	}

	cfg.RepoRoot, err = absFromConfig(path, cfg.RepoRoot)
	if err != nil {
		return cfg, err
	}
	cfg.WorkspaceRoot, err = absFromRepo(cfg.RepoRoot, cfg.WorkspaceRoot)
	if err != nil {
		return cfg, err
	}
	cfg.ArtifactRoot, err = absFromRepo(cfg.RepoRoot, cfg.ArtifactRoot)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func absFromConfig(configPath, p string) (string, error) {
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	// Configs usually live under experiments/configs; repo-relative paths are
	// resolved from the current process directory for predictable CLI use.
	return filepath.Abs(p)
}

func absFromRepo(repoRoot, p string) (string, error) {
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	return filepath.Abs(filepath.Join(repoRoot, p))
}

func prepare(cfg TrialConfig) error {
	statusf("prepare: trial=%s baseline=%s", cfg.TrialID, cfg.BaselineRef)
	if err := ensureCleanRepo(cfg.RepoRoot); err != nil {
		return err
	}

	sha, err := gitOutput(cfg.RepoRoot, "rev-parse", cfg.BaselineRef)
	if err != nil {
		return fmt.Errorf("resolve baseline_ref: %w", err)
	}
	sha = strings.TrimSpace(sha)

	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if exists(workspace) {
		return fmt.Errorf("trial workspace already exists: %s", workspace)
	}
	if exists(artifactDir) {
		return fmt.Errorf("artifact directory already exists: %s", artifactDir)
	}
	if err := os.MkdirAll(cfg.WorkspaceRoot, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	if _, err := gitOutput(cfg.RepoRoot, "worktree", "add", "--detach", workspace, sha); err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	statusf("prepare: created workspace %s", workspace)

	manifest := map[string]any{
		"trial_config": cfg,
		"baseline_sha": sha,
		"prepared_at":  time.Now().UTC(),
		"runner": map[string]string{
			"name": "salescraft-exp",
		},
		"environment": environmentSummary(),
	}
	if err := writeJSON(filepath.Join(artifactDir, "manifest.json"), manifest); err != nil {
		return err
	}

	digest, err := inputDigest(workspace)
	if err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(artifactDir, "input-digest.json"), digest); err != nil {
		return err
	}
	statusf("prepare: wrote artifacts %s", artifactDir)
	return nil
}

func runTool(cfg TrialConfig) error {
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	stdoutPath := filepath.Join(artifactDir, "tool-stdout.log")
	stderrPath := filepath.Join(artifactDir, "tool-stderr.log")
	statusf("run: trial=%s tool=%s command=%s", cfg.TrialID, cfg.Tool.Name, strings.Join(append([]string{cfg.Tool.Command}, cfg.Tool.Args...), " "))
	result, err := runCaptured(workspace, cfg.Tool.Command, cfg.Tool.Args, trialEnv(cfg, workspace, artifactDir), stdoutPath, stderrPath)
	writeErr := writeJSON(filepath.Join(artifactDir, "tool-result.json"), result)
	if err != nil {
		return fmt.Errorf("%w; see %s and %s", err, stdoutPath, stderrPath)
	}
	statusf("run: completed in %s", formatDuration(result.DurationMS))
	return writeErr
}

func verify(cfg TrialConfig) error {
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	logPath := filepath.Join(artifactDir, "verify-log.txt")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	var results []CommandResult
	for _, command := range cfg.Verification.Commands {
		if len(command) == 0 {
			continue
		}
		statusf("verify: running %s", strings.Join(command, " "))
		_, _ = fmt.Fprintf(logFile, "\n$ %s\n", strings.Join(command, " "))
		stdout := io.MultiWriter(logFile, os.Stdout)
		stderr := io.MultiWriter(logFile, os.Stderr)
		result := runWithWriters(workspace, command[0], command[1:], trialEnv(cfg, workspace, artifactDir), stdout, stderr)
		results = append(results, result)
		if result.ExitCode != 0 {
			_ = writeJSON(filepath.Join(artifactDir, "verify-result.json"), results)
			return fmt.Errorf("verification failed: %s", strings.Join(command, " "))
		}
		statusf("verify: passed in %s", formatDuration(result.DurationMS))
	}
	return writeJSON(filepath.Join(artifactDir, "verify-result.json"), results)
}

func archive(cfg TrialConfig) error {
	statusf("archive: trial=%s", cfg.TrialID)
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	status, _ := gitOutput(workspace, "status", "--short")
	diff, _ := gitOutput(workspace, "diff", "--binary")
	if err := os.WriteFile(filepath.Join(artifactDir, "final-status.txt"), []byte(status), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "final-diff.patch"), []byte(diff), 0o644); err != nil {
		return err
	}
	if err := tarGz(workspace, filepath.Join(artifactDir, "generated-repo.tar.gz")); err != nil {
		return err
	}
	statusf("archive: wrote %s", artifactDir)
	return nil
}

func clean(cfg TrialConfig) error {
	workspace := workspacePath(cfg)
	if !exists(workspace) {
		return nil
	}
	statusf("clean: removing workspace %s", workspace)
	if _, err := gitOutput(cfg.RepoRoot, "worktree", "remove", "--force", workspace); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}
	statusf("clean: removed workspace")
	return nil
}

func trial(cfg TrialConfig) error {
	if err := prepare(cfg); err != nil {
		return err
	}
	var runErr error
	if err := runTool(cfg); err != nil {
		runErr = err
	} else if err := verify(cfg); err != nil {
		runErr = err
	}
	archiveErr := archive(cfg)
	if runErr != nil {
		return runErr
	}
	if archiveErr != nil {
		return archiveErr
	}
	statusf("trial: workspace retained for manual testing: %s", workspacePath(cfg))
	statusf("trial: run clean explicitly when testing is complete")
	return nil
}

func runCaptured(workdir, command string, args []string, env map[string]string, stdoutPath, stderrPath string) (CommandResult, error) {
	stdout, err := os.Create(stdoutPath)
	if err != nil {
		return CommandResult{}, err
	}
	defer stdout.Close()
	stderr, err := os.Create(stderrPath)
	if err != nil {
		return CommandResult{}, err
	}
	defer stderr.Close()

	result := runWithWriters(workdir, command, args, env, io.MultiWriter(stdout, os.Stdout), io.MultiWriter(stderr, os.Stderr))
	if result.ExitCode != 0 {
		return result, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, strings.Join(result.Command, " "))
	}
	return result, nil
}

func runWithWriters(workdir, command string, args []string, env map[string]string, stdout, stderr io.Writer) CommandResult {
	start := time.Now().UTC()
	fullCommand := append([]string{command}, args...)
	statusf("command: start %s", strings.Join(fullCommand, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = workdir
	cmd.Env = mergedEnv(env)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Start()
	if err == nil {
		done := make(chan struct{})
		go heartbeat(done, start, fullCommand)
		err = cmd.Wait()
		close(done)
	}
	finish := time.Now().UTC()

	result := CommandResult{
		Command:    fullCommand,
		StartedAt:  start,
		FinishedAt: finish,
		DurationMS: finish.Sub(start).Milliseconds(),
		ExitCode:   0,
	}
	if err != nil {
		result.Error = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}
	if result.ExitCode == 0 {
		statusf("command: completed in %s", formatDuration(result.DurationMS))
	} else {
		statusf("command: failed after %s with exit code %d", formatDuration(result.DurationMS), result.ExitCode)
	}
	return result
}

func heartbeat(done <-chan struct{}, start time.Time, command []string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case now := <-ticker.C:
			statusf("command: still running after %s: %s", now.Sub(start).Round(time.Second), strings.Join(command, " "))
		}
	}
}

func statusf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[salescraft-exp] "+format+"\n", args...)
}

func formatDuration(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).Round(time.Millisecond).String()
}

func mergedEnv(extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

func trialEnv(cfg TrialConfig, workspace, artifactDir string) map[string]string {
	cacheRoot := filepath.Join(os.TempDir(), "salescraft-exp", cfg.TrialID)
	env := map[string]string{
		"SALESCRAFT_EXPERIMENT_RUNNER": "salescraft-exp",
		"SALESCRAFT_TRIAL_ID":          cfg.TrialID,
		"SALESCRAFT_WORKSPACE":         workspace,
		"SALESCRAFT_ARTIFACT_DIR":      artifactDir,
		"SALESCRAFT_CACHE_DIR":         cacheRoot,
		"COREPACK_HOME":                filepath.Join(cacheRoot, "corepack"),
		"NPM_CONFIG_CACHE":             filepath.Join(cacheRoot, "npm"),
		"npm_config_cache":             filepath.Join(cacheRoot, "npm"),
		"PNPM_HOME":                    filepath.Join(cacheRoot, "pnpm-home"),
		"XDG_CACHE_HOME":               filepath.Join(cacheRoot, "xdg"),
	}
	for k, v := range cfg.Tool.Env {
		env[k] = v
	}
	return env
}

func ensureCleanRepo(repoRoot string) error {
	out, err := gitOutput(repoRoot, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("golden repo has uncommitted changes; commit or stash before preparing a trial")
	}
	return nil
}

func gitOutput(workdir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	b, err := cmd.CombinedOutput()
	if err != nil {
		return string(b), fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(b))
	}
	return string(b), nil
}

func inputDigest(root string) (map[string]any, error) {
	files := []string{
		"PROMPT.md",
		"BUILD_PLAN.md",
		"AGENT_OPERATING_MODEL.md",
		"EVALUATION_PROTOCOL.md",
	}
	specs, err := filepath.Glob(filepath.Join(root, "spec", "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(specs)
	for _, spec := range specs {
		rel, err := filepath.Rel(root, spec)
		if err != nil {
			return nil, err
		}
		files = append(files, rel)
	}

	hashes := map[string]string{}
	for _, rel := range files {
		path := filepath.Join(root, rel)
		sum, err := fileSHA256(path)
		if err != nil {
			return nil, err
		}
		hashes[rel] = sum
	}
	return map[string]any{
		"generated_at": time.Now().UTC(),
		"files":        hashes,
	}, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func environmentSummary() map[string]string {
	keys := []string{
		"GOOS",
		"GOARCH",
		"SHELL",
		"TERM",
		"USER",
	}
	out := map[string]string{}
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			out[key] = val
		}
	}
	return out
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func tarGz(srcDir, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipArchive(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

func shouldSkipArchive(rel string, d os.DirEntry) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case ".git", "node_modules", ".next", "dist", "build", ".turbo", "coverage":
		return true
	default:
		return false
	}
}

func workspacePath(cfg TrialConfig) string {
	return filepath.Join(cfg.WorkspaceRoot, cfg.TrialID)
}

func artifactPath(cfg TrialConfig) string {
	return filepath.Join(cfg.ArtifactRoot, cfg.TrialID)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func exitOnErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
