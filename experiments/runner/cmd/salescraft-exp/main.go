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
	"regexp"
	"sort"
	"strconv"
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
	configPath, trialIDOverride := parseFlags(os.Args[2:])
	cfg, err := loadConfig(configPath)
	exitOnErr(err)
	if trialIDOverride != "" {
		cfg.TrialID = trialIDOverride
	}

	switch cmd {
	case "prepare":
		err = prepare(cfg)
	case "run":
		err = runTool(cfg)
	case "verify":
		err = verify(cfg)
	case "archive":
		err = archive(cfg)
	case "evaluate":
		err = evaluate(cfg)
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
	fmt.Fprintf(os.Stderr, "usage: salescraft-exp {prepare|run|verify|archive|evaluate|clean|trial} --config path/to/trial.json [--trial-id id]\n")
	os.Exit(2)
}

func parseFlags(args []string) (string, string) {
	fs := flag.NewFlagSet("salescraft-exp", flag.ExitOnError)
	configPath := fs.String("config", "", "trial config JSON path")
	trialID := fs.String("trial-id", "", "override trial_id from the config")
	_ = fs.Parse(args)
	if *configPath == "" {
		usageAndExit()
	}
	return *configPath, *trialID
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

	if _, err := gitOutput(cfg.RepoRoot, "clone", "--local", "--no-hardlinks", cfg.RepoRoot, workspace); err != nil {
		return fmt.Errorf("create trial clone: %w", err)
	}
	if _, err := gitOutput(workspace, "checkout", "--detach", sha); err != nil {
		return fmt.Errorf("checkout baseline in trial clone: %w", err)
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
	return runToolIteration(cfg, 0)
}

func runToolIteration(cfg TrialConfig, iteration int) error {
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	stdoutPath := filepath.Join(artifactDir, iterationFile("tool", iteration, "stdout.log"))
	stderrPath := filepath.Join(artifactDir, iterationFile("tool", iteration, "stderr.log"))
	env, err := trialEnv(cfg, workspace, artifactDir)
	if err != nil {
		return err
	}
	statusf("run: trial=%s tool=%s command=%s", cfg.TrialID, cfg.Tool.Name, strings.Join(append([]string{cfg.Tool.Command}, cfg.Tool.Args...), " "))
	result, err := runCaptured(workspace, cfg.Tool.Command, cfg.Tool.Args, env, stdoutPath, stderrPath)
	writeErr := writeJSON(filepath.Join(artifactDir, iterationFile("tool", iteration, "result.json")), result)
	if err != nil {
		return fmt.Errorf("%w; see %s and %s", err, stdoutPath, stderrPath)
	}
	statusf("run: completed in %s", formatDuration(result.DurationMS))
	return writeErr
}

func verify(cfg TrialConfig) error {
	return verifyIteration(cfg, 0)
}

type EvaluationResult struct {
	TrialID              string                 `json:"trial_id"`
	GeneratedAt          time.Time              `json:"generated_at"`
	Workspace            string                 `json:"workspace"`
	ArtifactDir          string                 `json:"artifact_dir"`
	Outcome              string                 `json:"outcome"`
	Tool                 string                 `json:"tool"`
	Phase                string                 `json:"phase"`
	AllowedVariable      string                 `json:"allowed_variable"`
	Models               map[string]string      `json:"models"`
	BuildComplete        bool                   `json:"build_complete"`
	BuildBlocked         bool                   `json:"build_blocked"`
	NextEligible         string                 `json:"next_eligible"`
	CompletedItems       []string               `json:"completed_items"`
	IterationCount       int                    `json:"iteration_count"`
	ToolDurationMS       int64                  `json:"tool_duration_ms"`
	TokenUsage           TokenUsage             `json:"token_usage"`
	RunnerVerifyPasses   int                    `json:"runner_verify_passes"`
	RunnerVerifyFailures int                    `json:"runner_verify_failures"`
	EvaluatorCommands    []CommandResult        `json:"evaluator_commands"`
	VerificationPassed   bool                   `json:"verification_passed"`
	EvaluatorPassed      bool                   `json:"evaluator_passed"`
	BrowserEvaluation    map[string]interface{} `json:"browser_evaluation,omitempty"`
	BrowserPassed        bool                   `json:"browser_passed"`
	DDDScore             map[string]interface{} `json:"ddd_score"`
	QualityScore         map[string]interface{} `json:"quality_score"`
	ResearchAnswers      map[string]string      `json:"research_answers"`
	JudgeReview          JudgeReview            `json:"judge_review"`
	FinalStatus          map[string]string      `json:"final_status"`
	ArtifactFiles        []string               `json:"artifact_files"`
	WorkspaceStats       map[string]int         `json:"workspace_stats"`
	PackageScripts       map[string][]string    `json:"package_scripts"`
	Notes                []string               `json:"notes,omitempty"`
	Environment          map[string]string      `json:"environment"`
	InputDigest          map[string]interface{} `json:"input_digest,omitempty"`
}

type JudgeReview struct {
	Available  bool       `json:"available"`
	Ran        bool       `json:"ran"`
	Passed     bool       `json:"passed"`
	Verdict    string     `json:"verdict,omitempty"`
	Model      string     `json:"model,omitempty"`
	Command    []string   `json:"command,omitempty"`
	StartedAt  time.Time  `json:"started_at,omitempty"`
	FinishedAt time.Time  `json:"finished_at,omitempty"`
	DurationMS int64      `json:"duration_ms,omitempty"`
	ExitCode   int        `json:"exit_code,omitempty"`
	Error      string     `json:"error,omitempty"`
	PromptPath string     `json:"prompt_path,omitempty"`
	ReportPath string     `json:"report_path,omitempty"`
	StdoutPath string     `json:"stdout_path,omitempty"`
	StderrPath string     `json:"stderr_path,omitempty"`
	TokenUsage TokenUsage `json:"token_usage,omitempty"`
	Notes      []string   `json:"notes,omitempty"`
}

type TokenUsage struct {
	Available   bool             `json:"available"`
	Total       int64            `json:"total"`
	ByIteration map[string]int64 `json:"by_iteration,omitempty"`
	Source      string           `json:"source,omitempty"`
	Notes       []string         `json:"notes,omitempty"`
}

func verifyIteration(cfg TrialConfig, iteration int) error {
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}

	logPath := filepath.Join(artifactDir, iterationFile("verify", iteration, "log.txt"))
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
		env, err := trialEnv(cfg, workspace, artifactDir)
		if err != nil {
			return err
		}
		result := runWithWriters(workspace, command[0], command[1:], env, stdout, stderr)
		results = append(results, result)
		if result.ExitCode != 0 {
			_ = writeJSON(filepath.Join(artifactDir, iterationFile("verify", iteration, "result.json")), results)
			return fmt.Errorf("verification failed: %s", strings.Join(command, " "))
		}
		statusf("verify: passed in %s", formatDuration(result.DurationMS))
	}
	return writeJSON(filepath.Join(artifactDir, iterationFile("verify", iteration, "result.json")), results)
}

func evaluate(cfg TrialConfig) error {
	statusf("evaluate: trial=%s", cfg.TrialID)
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	if !exists(workspace) {
		return fmt.Errorf("trial workspace does not exist: %s", workspace)
	}
	if !exists(artifactDir) {
		return fmt.Errorf("artifact directory does not exist: %s", artifactDir)
	}

	logPath := filepath.Join(artifactDir, "evaluation-log.txt")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	evalResults := runEvaluationCommands(cfg, workspace, artifactDir, logFile)
	browserEval := runBrowserEvaluation(cfg, workspace, artifactDir, logFile)
	report, err := buildEvaluationResult(cfg, evalResults, browserEval)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "final-report.md"), []byte(finalReportMarkdown(report)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "judge-brief.md"), []byte(judgeBriefMarkdown(report)), 0o644); err != nil {
		return err
	}
	report.JudgeReview = runJudgeEvaluation(cfg, report, logFile)
	report.EvaluatorPassed = report.EvaluatorPassed && report.JudgeReview.Passed
	report.ResearchAnswers = researchAnswers(report, cfg)
	if err := writeJSON(filepath.Join(artifactDir, "evaluation-result.json"), report); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "final-report.md"), []byte(finalReportMarkdown(report)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "judge-brief.md"), []byte(judgeBriefMarkdown(report)), 0o644); err != nil {
		return err
	}
	statusf("evaluate: wrote %s", filepath.Join(artifactDir, "final-report.md"))
	if !report.EvaluatorPassed {
		return fmt.Errorf("independent evaluation quality gate failed; see %s and %s", filepath.Join(artifactDir, "final-report.md"), logPath)
	}
	return nil
}

func runEvaluationCommands(cfg TrialConfig, workspace, artifactDir string, logFile io.Writer) []CommandResult {
	var results []CommandResult
	for _, command := range cfg.Verification.Commands {
		if len(command) == 0 {
			continue
		}
		statusf("evaluate: running %s", strings.Join(command, " "))
		_, _ = fmt.Fprintf(logFile, "\n$ %s\n", strings.Join(command, " "))
		stdout := io.MultiWriter(logFile, os.Stdout)
		stderr := io.MultiWriter(logFile, os.Stderr)
		env, err := trialEnv(cfg, workspace, artifactDir)
		if err != nil {
			results = append(results, CommandResult{
				Command:    command,
				StartedAt:  time.Now().UTC(),
				FinishedAt: time.Now().UTC(),
				ExitCode:   -1,
				Error:      err.Error(),
			})
			continue
		}
		result := runWithWriters(workspace, command[0], command[1:], env, stdout, stderr)
		results = append(results, result)
	}
	return results
}

func runBrowserEvaluation(cfg TrialConfig, workspace, artifactDir string, logFile io.Writer) map[string]interface{} {
	scriptPath := filepath.Join(cfg.RepoRoot, "experiments", "scripts", "evaluate-app-ui.mjs")
	result := map[string]interface{}{
		"passed": false,
		"notes":  []string{},
	}
	if !exists(scriptPath) {
		result["failures"] = []string{"browser evaluator script is missing: " + scriptPath}
		return result
	}
	statusf("evaluate: running browser workflow evaluator")
	_, _ = fmt.Fprintf(logFile, "\n$ node %s %s %s\n", scriptPath, workspace, artifactDir)
	env, err := trialEnv(cfg, workspace, artifactDir)
	if err != nil {
		result["failures"] = []string{err.Error()}
		return result
	}
	cmd := exec.Command("node", scriptPath, workspace, artifactDir)
	cmd.Dir = workspace
	cmd.Env = mergedEnv(env)
	out, err := cmd.CombinedOutput()
	_, _ = logFile.Write(out)
	if err != nil {
		result["command_error"] = err.Error()
	}
	if len(out) > 0 {
		if parseErr := json.Unmarshal(out, &result); parseErr != nil {
			result["parse_error"] = parseErr.Error()
			result["raw_output"] = string(out)
		}
	}
	_ = writeJSON(filepath.Join(artifactDir, "browser-evaluation.json"), result)
	return result
}

func runJudgeEvaluation(cfg TrialConfig, report EvaluationResult, logFile io.Writer) JudgeReview {
	artifactDir := report.ArtifactDir
	model := strings.TrimSpace(cfg.Models["judge"])
	review := JudgeReview{
		Available:  false,
		Ran:        false,
		Passed:     false,
		Model:      model,
		PromptPath: filepath.Join(artifactDir, "judge-prompt.md"),
		ReportPath: filepath.Join(artifactDir, "judge-report.md"),
		StdoutPath: filepath.Join(artifactDir, "judge-stdout.log"),
		StderrPath: filepath.Join(artifactDir, "judge-stderr.log"),
	}
	if model == "" {
		review.Notes = append(review.Notes, "models.judge is not configured")
		return review
	}
	if !strings.HasPrefix(model, "openai.") {
		review.Notes = append(review.Notes, "Codex judge execution is only enabled for OpenAI model IDs; configured judge model is "+model)
		return review
	}
	if _, err := exec.LookPath("codex"); err != nil {
		review.Notes = append(review.Notes, "codex executable not found: "+err.Error())
		return review
	}

	prompt := judgePrompt(report)
	if err := os.WriteFile(review.PromptPath, []byte(prompt), 0o644); err != nil {
		review.Error = "write judge prompt: " + err.Error()
		return review
	}

	args := []string{
		"exec",
		"--disable",
		"tui_app_server",
		"-c",
		"model_provider=\"amazon-bedrock\"",
		"-c",
		"shell_environment_policy.inherit=\"all\"",
		"--model",
		model,
	}
	review.Available = true
	review.Ran = true
	review.Command = append([]string{"codex"}, args...)
	review.StartedAt = time.Now().UTC()
	statusf("evaluate: running LLM judge model=%s", model)
	_, _ = fmt.Fprintf(logFile, "\n$ codex %s < %s\n", strings.Join(args, " "), review.PromptPath)

	stdoutFile, stdoutErr := os.Create(review.StdoutPath)
	if stdoutErr != nil {
		review.Error = "create judge stdout log: " + stdoutErr.Error()
		return review
	}
	defer stdoutFile.Close()
	stderrFile, stderrErr := os.Create(review.StderrPath)
	if stderrErr != nil {
		review.Error = "create judge stderr log: " + stderrErr.Error()
		return review
	}
	defer stderrFile.Close()

	cmd := exec.Command("codex", args...)
	cmd.Dir = artifactDir
	env, envErr := trialEnv(cfg, report.Workspace, artifactDir)
	if envErr != nil {
		review.Error = envErr.Error()
		return review
	}
	cmd.Env = mergedEnv(env)
	cmd.Stdin = strings.NewReader(prompt)
	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	cmd.Stdout = io.MultiWriter(stdoutFile, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(stderrFile, &stderrBuf, logFile)
	err := cmd.Run()
	review.FinishedAt = time.Now().UTC()
	review.DurationMS = review.FinishedAt.Sub(review.StartedAt).Milliseconds()
	if cmd.ProcessState != nil {
		review.ExitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		review.ExitCode = -1
	}
	if err != nil {
		review.Error = err.Error()
	}

	judgeOutput := strings.TrimSpace(stdoutBuf.String())
	if judgeOutput == "" {
		review.Notes = append(review.Notes, "judge produced no stdout")
	} else if writeErr := os.WriteFile(review.ReportPath, []byte(judgeOutput+"\n"), 0o644); writeErr != nil {
		review.Notes = append(review.Notes, "could not write judge report: "+writeErr.Error())
	}
	review.Verdict = parseJudgeVerdict(judgeOutput)
	review.Passed = review.ExitCode == 0 && review.Verdict == "pass"
	review.TokenUsage = tokenUsageFromText(stderrBuf.String(), "codex judge stderr 'tokens used' summary")
	if review.Verdict == "" {
		review.Notes = append(review.Notes, "judge report did not include a parseable first-line verdict")
	}
	_ = writeJSON(filepath.Join(artifactDir, "judge-result.json"), review)
	return review
}

func judgePrompt(report EvaluationResult) string {
	var b strings.Builder
	b.WriteString("You are the independent LLM judge for a Salescraft autonomous SDLC experiment.\n")
	b.WriteString("This is a read-only evaluation step. Do not modify files. Do not run commands. Use only the evidence pasted below.\n\n")
	b.WriteString("Your first line must be exactly one of:\n")
	b.WriteString("Verdict: pass\nVerdict: marginal\nVerdict: fail\n\n")
	b.WriteString("Use `pass` only if the generated app satisfies the research-quality acceptance bar. Use `marginal` when it is useful but incomplete. Use `fail` when critical acceptance surfaces are missing or the evidence is insufficient.\n\n")
	b.WriteString("After the verdict, include concise sections:\n")
	b.WriteString("- Functional correctness\n- DDD/domain critique\n- Code and test quality\n- Research implications for RQ0, RQ2, RQ4, RQ8, RQ9, and RQ10\n- Data gaps\n\n")
	b.WriteString("## Judge Brief\n\n")
	b.WriteString(judgeBriefMarkdown(report))
	b.WriteString("\n\n## Deterministic Evaluation Summary\n\n")
	b.WriteString(fmt.Sprintf("- Quality gate before judge: `%t`\n", report.EvaluatorPassed))
	b.WriteString(fmt.Sprintf("- Fixed verification passed: `%t`\n", report.VerificationPassed))
	b.WriteString(fmt.Sprintf("- Browser/workflow checks passed: `%t`\n", report.BrowserPassed))
	b.WriteString(fmt.Sprintf("- Quality score: `%0.1f/100`\n", floatFromMap(report.QualityScore, "total")))
	b.WriteString(fmt.Sprintf("- Static DDD score: `%0.0f/100`\n", floatFromMap(report.DDDScore, "score")))
	b.WriteString(fmt.Sprintf("- Tool iterations: `%d`\n", report.IterationCount))
	b.WriteString(fmt.Sprintf("- Tool runtime: `%s`\n", formatDuration(report.ToolDurationMS)))
	b.WriteString(fmt.Sprintf("- Observed tool tokens: `%s`\n", formatInt(report.TokenUsage.Total)))
	b.WriteString(fmt.Sprintf("- Completed build-plan items: `%d`\n", len(report.CompletedItems)))
	b.WriteString("\nCritical findings from deterministic evaluator:\n")
	if !report.VerificationPassed {
		b.WriteString("- Fixed verification failed.\n")
	}
	if !report.BrowserPassed {
		failures := failedBrowserRoutes(report.BrowserEvaluation)
		if len(failures) > 0 {
			b.WriteString("- Browser evaluation found broken or incomplete user-facing routes: ")
			b.WriteString(strings.Join(failures, ", "))
			b.WriteString(".\n")
		} else {
			b.WriteString("- One or more browser workflow checks failed.\n")
		}
	}
	missingTerms := stringSliceFromMap(report.DDDScore, "missing_terms")
	if len(missingTerms) > 0 {
		b.WriteString("- Missing required commercial flooring domain language: ")
		b.WriteString(strings.Join(missingTerms, ", "))
		b.WriteString(".\n")
	}
	if browserJSON, err := os.ReadFile(filepath.Join(report.ArtifactDir, "browser-evaluation.json")); err == nil {
		b.WriteString("\n\n## Browser Evaluation JSON\n\n```json\n")
		b.Write(browserJSON)
		b.WriteString("\n```\n")
	}
	if rubric, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(report.ArtifactDir)), "evaluation", "research-rubric.md")); err == nil {
		b.WriteString("\n\n## Research Rubric\n\n")
		b.Write(rubric)
	}
	return b.String()
}

func parseJudgeVerdict(output string) string {
	verdictPattern := regexp.MustCompile(`(?im)^\s*Verdict:\s*(pass|marginal|fail)\s*$`)
	match := verdictPattern.FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func buildEvaluationResult(cfg TrialConfig, evalResults []CommandResult, browserEval map[string]interface{}) (EvaluationResult, error) {
	workspace := workspacePath(cfg)
	artifactDir := artifactPath(cfg)
	state, stateErr := readBuildState(workspace)
	finalStatus, _ := readKeyValueFile(filepath.Join(artifactDir, "final-status.txt"))
	artifactFiles, _ := artifactFileList(artifactDir)
	toolResults, _ := commandResultsFromGlob(filepath.Join(artifactDir, "tool-iteration-*-result.json"))
	verifyResults, _ := verificationSummary(artifactDir)
	completedItems := []string{}
	if b, err := os.ReadFile(filepath.Join(workspace, "BUILD_STATE.md")); err == nil {
		completedItems = sectionItems(string(b), "## Completed")
	}
	inputDigest, _ := readJSONMap(filepath.Join(artifactDir, "input-digest.json"))
	dddScore := dddEvaluation(workspace)
	browserPassed := boolFromMap(browserEval, "passed")
	verificationPassed := allCommandsPassed(evalResults)
	tokenUsage := parseTokenUsage(artifactDir)

	report := EvaluationResult{
		TrialID:              cfg.TrialID,
		GeneratedAt:          time.Now().UTC(),
		Workspace:            workspace,
		ArtifactDir:          artifactDir,
		Outcome:              firstNonEmpty(finalStatus["outcome"], "unknown"),
		Tool:                 cfg.Tool.Name,
		Phase:                cfg.Phase,
		AllowedVariable:      cfg.AllowedVariable,
		Models:               cfg.Models,
		CompletedItems:       completedItems,
		IterationCount:       len(toolResults),
		ToolDurationMS:       sumCommandDurations(toolResults),
		TokenUsage:           tokenUsage,
		RunnerVerifyPasses:   verifyResults.Passes,
		RunnerVerifyFailures: verifyResults.Failures,
		EvaluatorCommands:    evalResults,
		VerificationPassed:   verificationPassed,
		EvaluatorPassed:      verificationPassed && browserPassed,
		BrowserEvaluation:    browserEval,
		BrowserPassed:        browserPassed,
		DDDScore:             dddScore,
		FinalStatus:          finalStatus,
		ArtifactFiles:        artifactFiles,
		WorkspaceStats:       workspaceStats(workspace),
		PackageScripts:       packageScripts(workspace),
		Environment:          environmentSummary(),
		InputDigest:          inputDigest,
	}
	report.QualityScore = qualityScore(report)
	report.ResearchAnswers = researchAnswers(report, cfg)
	if stateErr == nil {
		report.BuildComplete = state.Complete
		report.BuildBlocked = state.Blocked
		report.NextEligible = state.NextEligibleSummary
	} else {
		report.Notes = append(report.Notes, "BUILD_STATE.md could not be read: "+stateErr.Error())
	}
	if report.Outcome == "unknown" {
		report.Notes = append(report.Notes, "final-status.txt missing or does not include an outcome")
	}
	return report, nil
}

func dddEvaluation(workspace string) map[string]interface{} {
	requiredTerms := []string{
		"specification",
		"takeoff",
		"wear layer",
		"seaming",
		"transition strip",
		"punch list",
		"general contractor",
		"architect of record",
		"material safety data sheet",
		"msds",
		"lead time",
		"floor prep",
		"attic stock",
	}
	boundedContexts := []string{
		"sales",
		"product",
		"project",
		"marketing",
		"recommendation",
		"intelligence",
	}
	genericTerms := []string{"todo app", "users table", "items controller", "generic crud"}
	text := strings.ToLower(codeCorpus(workspace))
	presentTerms := []string{}
	missingTerms := []string{}
	for _, term := range requiredTerms {
		if strings.Contains(text, term) {
			presentTerms = append(presentTerms, term)
		} else {
			missingTerms = append(missingTerms, term)
		}
	}
	presentContexts := []string{}
	for _, context := range boundedContexts {
		if strings.Contains(text, context) {
			presentContexts = append(presentContexts, context)
		}
	}
	genericHits := []string{}
	for _, term := range genericTerms {
		if strings.Contains(text, term) {
			genericHits = append(genericHits, term)
		}
	}
	termRatio := float64(len(presentTerms)) / float64(len(requiredTerms))
	contextRatio := float64(len(presentContexts)) / float64(len(boundedContexts))
	score := int((termRatio*0.6 + contextRatio*0.4) * 100)
	score -= len(genericHits) * 5
	if score < 0 {
		score = 0
	}
	return map[string]interface{}{
		"score":            score,
		"present_terms":    presentTerms,
		"missing_terms":    missingTerms,
		"present_contexts": presentContexts,
		"generic_hits":     genericHits,
	}
}

func codeCorpus(workspace string) string {
	var b strings.Builder
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(workspace, path)
		if relErr == nil && shouldSkipEvaluationPath(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(d.Name()) {
		case ".ts", ".tsx", ".js", ".jsx", ".md", ".json", ".sql", ".prisma":
			content, readErr := os.ReadFile(path)
			if readErr == nil {
				b.Write(content)
				b.WriteString("\n")
			}
		}
		return nil
	})
	return b.String()
}

func parseTokenUsage(artifactDir string) TokenUsage {
	usage := TokenUsage{
		Available:   false,
		ByIteration: map[string]int64{},
		Source:      "codex stderr 'tokens used' summary",
	}
	paths, err := filepath.Glob(filepath.Join(artifactDir, "tool-iteration-*-stderr.log"))
	if err != nil {
		usage.Notes = append(usage.Notes, "could not glob tool stderr logs: "+err.Error())
		return usage
	}
	sort.Strings(paths)
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			usage.Notes = append(usage.Notes, "could not read "+filepath.Base(path)+": "+readErr.Error())
			continue
		}
		fileUsage := tokenUsageFromText(string(content), usage.Source)
		if !fileUsage.Available {
			continue
		}
		iteration := strings.TrimSuffix(filepath.Base(path), "-stderr.log")
		iteration = strings.TrimPrefix(iteration, "tool-iteration-")
		value := fileUsage.Total
		usage.ByIteration[iteration] = value
		usage.Total += value
		usage.Available = true
	}
	if !usage.Available {
		usage.Notes = append(usage.Notes, "no Codex token summary lines were found in tool stderr logs")
	}
	return usage
}

func tokenUsageFromText(text, source string) TokenUsage {
	usage := TokenUsage{
		Available: false,
		Source:    source,
	}
	tokenPattern := regexp.MustCompile(`(?m)tokens used\s*\r?\n\s*([0-9][0-9,]*)`)
	matches := tokenPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		usage.Notes = append(usage.Notes, "no token summary line found")
		return usage
	}
	last := matches[len(matches)-1][1]
	value, err := strconv.ParseInt(strings.ReplaceAll(last, ",", ""), 10, 64)
	if err != nil {
		usage.Notes = append(usage.Notes, "could not parse token count: "+err.Error())
		return usage
	}
	usage.Available = true
	usage.Total = value
	return usage
}

func qualityScore(report EvaluationResult) map[string]interface{} {
	functional := 0.0
	if report.VerificationPassed {
		functional += 22
	}
	if report.BrowserPassed {
		functional += 13
	}
	ddd := floatFromMap(report.DDDScore, "score") * 0.20
	codeQuality := 0.0
	if report.RunnerVerifyFailures == 0 && report.VerificationPassed {
		codeQuality = 15
	}
	completeness := float64(len(report.CompletedItems)) / 47.0 * 15.0
	if completeness > 15 {
		completeness = 15
	}
	security := 3.0
	documentation := 3.0
	if exists(filepath.Join(report.Workspace, "README.md")) {
		documentation = 4.0
	}
	performance := browserPerformanceScore(report.BrowserEvaluation)
	total := functional + ddd + codeQuality + completeness + security + documentation + performance
	if total > 100 {
		total = 100
	}
	return map[string]interface{}{
		"total":                  round1(total),
		"functional_correctness": round1(functional),
		"ddd_adherence":          round1(ddd),
		"code_quality":           round1(codeQuality),
		"completeness":           round1(completeness),
		"security":               round1(security),
		"documentation":          round1(documentation),
		"performance":            round1(performance),
		"notes": []string{
			"Security score is provisional until CodeQL/Semgrep are added.",
			"Documentation score is provisional until LLM judge review is added.",
		},
	}
}

func researchAnswers(report EvaluationResult, cfg TrialConfig) map[string]string {
	quality := fmt.Sprintf("%.1f", floatFromMap(report.QualityScore, "total"))
	browserSummary := "browser workflow passed"
	if !report.BrowserPassed {
		browserSummary = "browser workflow failed"
	}
	judgeSummary := "LLM judge unavailable"
	if report.JudgeReview.Ran {
		judgeSummary = "LLM judge verdict=" + firstNonEmpty(report.JudgeReview.Verdict, "unparseable")
	}
	return map[string]string{
		"RQ0":  fmt.Sprintf("This trial provides one point on the model/toolchain frontier: quality score %s with tool %s, %s observed tokens, %s, and models %s.", quality, cfg.Tool.Name, formatInt(report.TokenUsage.Total), judgeSummary, mapSummary(cfg.Models)),
		"RQ2":  fmt.Sprintf("Code generation reached %d completed build-plan items in %d iterations with %d runner verification failures; fixed verification passed=%t and %s.", len(report.CompletedItems), report.IterationCount, report.RunnerVerifyFailures, report.VerificationPassed, browserSummary),
		"RQ4":  fmt.Sprintf("The orchestrator/toolchain under test was %s; it converged in %d iterations and the independent evaluator quality gate passed=%t.", cfg.Tool.Name, report.IterationCount, report.EvaluatorPassed),
		"RQ8":  fmt.Sprintf("Cost-quality scoring has observed token usage (%s tokens), but USD cost remains incomplete until provider pricing/rate capture is added.", formatInt(report.TokenUsage.Total)),
		"RQ9":  fmt.Sprintf("The loop strategy was %v with %d observed model iterations.", cfg.Loop["strategy"], report.IterationCount),
		"RQ10": fmt.Sprintf("DDD static score is %.0f/100; missing terms and bounded-context evidence are recorded in evaluation-result.json.", floatFromMap(report.DDDScore, "score")),
	}
}

func boolFromMap(values map[string]interface{}, key string) bool {
	raw, ok := values[key]
	if !ok {
		return false
	}
	value, ok := raw.(bool)
	return ok && value
}

func floatFromMap(values map[string]interface{}, key string) float64 {
	raw, ok := values[key]
	if !ok {
		return 0
	}
	switch value := raw.(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case json.Number:
		f, _ := value.Float64()
		return f
	default:
		return 0
	}
}

func browserPerformanceScore(browserEval map[string]interface{}) float64 {
	pages, ok := browserEval["pages"].([]interface{})
	if !ok || len(pages) == 0 {
		return 0
	}
	var total float64
	var count float64
	for _, raw := range pages {
		page, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		total += floatFromMap(page, "duration_ms")
		count++
	}
	if count == 0 {
		return 0
	}
	avg := total / count
	switch {
	case avg <= 1000:
		return 5
	case avg <= 2000:
		return 4
	case avg <= 4000:
		return 3
	default:
		return 2
	}
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func formatInt(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	raw := strconv.FormatInt(value, 10)
	if len(raw) <= 3 {
		return sign + raw
	}
	var b strings.Builder
	remainder := len(raw) % 3
	if remainder == 0 {
		remainder = 3
	}
	b.WriteString(raw[:remainder])
	for i := remainder; i < len(raw); i += 3 {
		b.WriteString(",")
		b.WriteString(raw[i : i+3])
	}
	return sign + b.String()
}

func mapSummary(values map[string]string) string {
	parts := []string{}
	for _, key := range sortedMapKeysString(values) {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, ", ")
}

func stringSliceFromMap(values map[string]interface{}, key string) []string {
	raw, ok := values[key]
	if !ok {
		return []string{}
	}
	switch items := raw.(type) {
	case []string:
		return items
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{}
	}
}

type verifySummary struct {
	Passes   int
	Failures int
}

func verificationSummary(artifactDir string) (verifySummary, error) {
	paths, err := filepath.Glob(filepath.Join(artifactDir, "verify-iteration-*-result.json"))
	if err != nil {
		return verifySummary{}, err
	}
	sort.Strings(paths)
	var summary verifySummary
	for _, path := range paths {
		var results []CommandResult
		if err := readJSONFile(path, &results); err != nil {
			continue
		}
		for _, result := range results {
			if result.ExitCode == 0 {
				summary.Passes++
			} else {
				summary.Failures++
			}
		}
	}
	return summary, nil
}

func commandResultsFromGlob(pattern string) ([]CommandResult, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var results []CommandResult
	for _, path := range paths {
		var result CommandResult
		if err := readJSONFile(path, &result); err == nil {
			results = append(results, result)
		}
	}
	return results, nil
}

func sumCommandDurations(results []CommandResult) int64 {
	var total int64
	for _, result := range results {
		total += result.DurationMS
	}
	return total
}

func allCommandsPassed(results []CommandResult) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		if result.ExitCode != 0 {
			return false
		}
	}
	return true
}

func artifactFileList(artifactDir string) ([]string, error) {
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}

func workspaceStats(workspace string) map[string]int {
	stats := map[string]int{
		"files_total":        0,
		"source_files":       0,
		"test_files":         0,
		"package_json_files": 0,
	}
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(workspace, path)
		if relErr == nil && shouldSkipEvaluationPath(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		stats["files_total"]++
		name := d.Name()
		ext := filepath.Ext(name)
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			stats["source_files"]++
		}
		if strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
			stats["test_files"]++
		}
		if name == "package.json" {
			stats["package_json_files"]++
		}
		return nil
	})
	return stats
}

func shouldSkipEvaluationPath(rel string, d os.DirEntry) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		switch part {
		case ".git", "node_modules", ".next", "dist", "build", ".turbo", "coverage":
			return true
		}
	}
	return false
}

func packageScripts(workspace string) map[string][]string {
	scripts := map[string][]string{}
	_ = filepath.WalkDir(workspace, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(workspace, path)
		if relErr == nil && shouldSkipEvaluationPath(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || d.Name() != "package.json" {
			return nil
		}
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if err := readJSONFile(path, &pkg); err != nil {
			return nil
		}
		names := make([]string, 0, len(pkg.Scripts))
		for name := range pkg.Scripts {
			names = append(names, name)
		}
		sort.Strings(names)
		if relErr == nil {
			scripts[filepath.ToSlash(rel)] = names
		}
		return nil
	})
	return scripts
}

func finalReportMarkdown(report EvaluationResult) string {
	var b strings.Builder
	b.WriteString("# Experiment Final Report\n\n")
	b.WriteString(fmt.Sprintf("- Trial ID: `%s`\n", report.TrialID))
	b.WriteString(fmt.Sprintf("- Outcome: `%s`\n", report.Outcome))
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", report.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Tool: `%s`\n", report.Tool))
	b.WriteString(fmt.Sprintf("- Phase: `%s`\n", report.Phase))
	b.WriteString(fmt.Sprintf("- Allowed variable: `%s`\n", report.AllowedVariable))
	b.WriteString(fmt.Sprintf("- Workspace: `%s`\n", report.Workspace))
	b.WriteString(fmt.Sprintf("- Artifacts: `%s`\n\n", report.ArtifactDir))

	b.WriteString("## Evaluator Verdict\n\n")
	b.WriteString(fmt.Sprintf("- Quality gate passed: `%t`\n", report.EvaluatorPassed))
	b.WriteString(fmt.Sprintf("- Fixed verification passed: `%t`\n", report.VerificationPassed))
	b.WriteString(fmt.Sprintf("- Browser/workflow checks passed: `%t`\n", report.BrowserPassed))
	b.WriteString(fmt.Sprintf("- LLM judge passed: `%t`\n", report.JudgeReview.Passed))
	b.WriteString(fmt.Sprintf("- Quality score: `%0.1f/100`\n", floatFromMap(report.QualityScore, "total")))
	b.WriteString(fmt.Sprintf("- DDD score: `%0.0f/100`\n\n", floatFromMap(report.DDDScore, "score")))

	findings := criticalFindings(report)
	if len(findings) > 0 {
		b.WriteString("## Critical Findings\n\n")
		for _, finding := range findings {
			b.WriteString(fmt.Sprintf("- %s\n", finding))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Research Question Summary\n\n")
	for _, key := range sortedMapKeysString(report.ResearchAnswers) {
		b.WriteString(fmt.Sprintf("- **%s:** %s\n", key, report.ResearchAnswers[key]))
	}
	b.WriteString("\n")

	b.WriteString("## LLM Judge Review\n\n")
	b.WriteString(fmt.Sprintf("- Judge available: `%t`\n", report.JudgeReview.Available))
	b.WriteString(fmt.Sprintf("- Judge ran: `%t`\n", report.JudgeReview.Ran))
	b.WriteString(fmt.Sprintf("- Judge model: `%s`\n", report.JudgeReview.Model))
	b.WriteString(fmt.Sprintf("- Judge verdict: `%s`\n", firstNonEmpty(report.JudgeReview.Verdict, "unavailable")))
	b.WriteString(fmt.Sprintf("- Judge passed: `%t`\n", report.JudgeReview.Passed))
	if report.JudgeReview.ReportPath != "" {
		b.WriteString(fmt.Sprintf("- Judge report: `%s`\n", report.JudgeReview.ReportPath))
	}
	if report.JudgeReview.DurationMS > 0 {
		b.WriteString(fmt.Sprintf("- Judge duration: `%s`\n", formatDuration(report.JudgeReview.DurationMS)))
	}
	if report.JudgeReview.TokenUsage.Available {
		b.WriteString(fmt.Sprintf("- Judge observed tokens: `%s`\n", formatInt(report.JudgeReview.TokenUsage.Total)))
	}
	if report.JudgeReview.Error != "" {
		b.WriteString(fmt.Sprintf("- Judge error: `%s`\n", report.JudgeReview.Error))
	}
	if len(report.JudgeReview.Notes) > 0 {
		for _, note := range report.JudgeReview.Notes {
			b.WriteString(fmt.Sprintf("- Judge note: %s\n", note))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Quality Score\n\n")
	b.WriteString(fmt.Sprintf("- Total: `%0.1f/100`\n", floatFromMap(report.QualityScore, "total")))
	for _, key := range []string{"functional_correctness", "ddd_adherence", "code_quality", "completeness", "security", "documentation", "performance"} {
		b.WriteString(fmt.Sprintf("- %s: `%0.1f`\n", strings.ReplaceAll(key, "_", " "), floatFromMap(report.QualityScore, key)))
	}
	b.WriteString("\n")

	b.WriteString("## Build State\n\n")
	b.WriteString(fmt.Sprintf("- Build complete: `%t`\n", report.BuildComplete))
	b.WriteString(fmt.Sprintf("- Build blocked: `%t`\n", report.BuildBlocked))
	b.WriteString(fmt.Sprintf("- Next eligible: `%s`\n", report.NextEligible))
	b.WriteString(fmt.Sprintf("- Completed items: `%d`\n\n", len(report.CompletedItems)))

	b.WriteString("## Verification\n\n")
	b.WriteString(fmt.Sprintf("- Runner verification command passes: `%d`\n", report.RunnerVerifyPasses))
	b.WriteString(fmt.Sprintf("- Runner verification command failures: `%d`\n", report.RunnerVerifyFailures))
	b.WriteString(fmt.Sprintf("- Fixed verification commands passed: `%t`\n", report.VerificationPassed))
	b.WriteString(fmt.Sprintf("- Independent evaluator passed: `%t`\n\n", report.EvaluatorPassed))
	b.WriteString("| Command | Exit | Duration |\n")
	b.WriteString("| --- | ---: | ---: |\n")
	for _, result := range report.EvaluatorCommands {
		b.WriteString(fmt.Sprintf("| `%s` | `%d` | `%s` |\n", strings.Join(result.Command, " "), result.ExitCode, formatDuration(result.DurationMS)))
	}
	b.WriteString("\n")

	b.WriteString("## Browser Functionality\n\n")
	b.WriteString(fmt.Sprintf("- Browser workflow passed: `%t`\n", report.BrowserPassed))
	if screenshotDir := strings.TrimSpace(fmt.Sprint(report.BrowserEvaluation["screenshot_dir"])); screenshotDir != "" {
		b.WriteString(fmt.Sprintf("- Browser screenshots: `%s`\n", screenshotDir))
	}
	if pages, ok := report.BrowserEvaluation["pages"].([]interface{}); ok {
		b.WriteString("\n| Route | Status | Duration | Result |\n")
		b.WriteString("| --- | ---: | ---: | --- |\n")
		for _, raw := range pages {
			page, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			route := fmt.Sprint(page["path"])
			status := int(floatFromMap(page, "status"))
			duration := formatDuration(int64(floatFromMap(page, "duration_ms")))
			result := "pass"
			if errors, ok := page["errors"].([]interface{}); ok && len(errors) > 0 {
				result = "fail"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%d` | `%s` | `%s` |\n", route, status, duration, result))
		}
	}
	if workflows, ok := report.BrowserEvaluation["workflows"].([]interface{}); ok {
		b.WriteString("\n| Workflow | Duration | Result | Failed Assertions |\n")
		b.WriteString("| --- | ---: | --- | --- |\n")
		for _, raw := range workflows {
			workflow, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			name := fmt.Sprint(workflow["name"])
			duration := formatDuration(int64(floatFromMap(workflow, "duration_ms")))
			passed := boolFromMap(workflow, "passed")
			result := "pass"
			if !passed {
				result = "fail"
			}
			failures := stringSliceFromMap(workflow, "errors")
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` |\n", name, duration, result, strings.Join(failures, "; ")))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Evaluator Coverage\n\n")
	b.WriteString("- Browser tests run headless through Playwright, so no visible browser window is opened during evaluation.\n")
	b.WriteString("- Browser coverage currently checks route availability, expected page text, basic interactive controls, shell navigation, login form basics, estimate builder surface, relationship intelligence surface, and bid response surface.\n")
	b.WriteString("- Quality scoring is a deterministic composite of fixed verification, browser checks, static DDD/domain-language scanning, completion count, and provisional security/documentation/performance heuristics.\n")
	b.WriteString("- The LLM judge is a read-only critique layer over the generated evidence packet; it is not allowed to modify the trial workspace.\n")
	b.WriteString("- The current evaluator is not yet a full manual QA substitute, deep code review, security audit, or accessibility audit.\n\n")

	b.WriteString("## Domain-Driven Design\n\n")
	b.WriteString(fmt.Sprintf("- Static DDD score: `%0.0f/100`\n", floatFromMap(report.DDDScore, "score")))
	b.WriteString(fmt.Sprintf("- Present domain terms: `%s`\n", strings.Join(stringSliceFromMap(report.DDDScore, "present_terms"), "`, `")))
	b.WriteString(fmt.Sprintf("- Missing domain terms: `%s`\n", strings.Join(stringSliceFromMap(report.DDDScore, "missing_terms"), "`, `")))
	b.WriteString(fmt.Sprintf("- Present bounded-context signals: `%s`\n\n", strings.Join(stringSliceFromMap(report.DDDScore, "present_contexts"), "`, `")))

	b.WriteString("## Iterations And Timing\n\n")
	b.WriteString(fmt.Sprintf("- Tool iterations: `%d`\n", report.IterationCount))
	b.WriteString(fmt.Sprintf("- Tool runtime total: `%s`\n\n", formatDuration(report.ToolDurationMS)))

	b.WriteString("## Token Usage\n\n")
	b.WriteString(fmt.Sprintf("- Token usage available: `%t`\n", report.TokenUsage.Available))
	b.WriteString(fmt.Sprintf("- Total observed tokens: `%s`\n", formatInt(report.TokenUsage.Total)))
	b.WriteString(fmt.Sprintf("- Source: `%s`\n", report.TokenUsage.Source))
	if len(report.TokenUsage.ByIteration) > 0 {
		b.WriteString("\n| Iteration | Tokens |\n")
		b.WriteString("| ---: | ---: |\n")
		for _, iteration := range sortedMapKeysInt64(report.TokenUsage.ByIteration) {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", iteration, formatInt(report.TokenUsage.ByIteration[iteration])))
		}
	}
	if len(report.TokenUsage.Notes) > 0 {
		b.WriteString("\n")
		for _, note := range report.TokenUsage.Notes {
			b.WriteString(fmt.Sprintf("- %s\n", note))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Workspace Inventory\n\n")
	keys := sortedMapKeysInt(report.WorkspaceStats)
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("- %s: `%d`\n", key, report.WorkspaceStats[key]))
	}
	b.WriteString("\n")

	b.WriteString("## Package Scripts\n\n")
	pkgKeys := sortedMapKeysStringSlice(report.PackageScripts)
	for _, key := range pkgKeys {
		b.WriteString(fmt.Sprintf("- `%s`: `%s`\n", key, strings.Join(report.PackageScripts[key], "`, `")))
	}
	b.WriteString("\n")

	b.WriteString("## Models\n\n")
	modelKeys := sortedMapKeysString(report.Models)
	for _, key := range modelKeys {
		b.WriteString(fmt.Sprintf("- %s: `%s`\n", key, report.Models[key]))
	}
	b.WriteString("\n")

	risks := residualRisks(report)
	if len(risks) > 0 {
		b.WriteString("## Residual Risks\n\n")
		for _, risk := range risks {
			b.WriteString(fmt.Sprintf("- %s\n", risk))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Artifact Files\n\n")
	for _, file := range report.ArtifactFiles {
		b.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	if len(report.Notes) > 0 {
		b.WriteString("\n## Notes\n\n")
		for _, note := range report.Notes {
			b.WriteString(fmt.Sprintf("- %s\n", note))
		}
	}
	return b.String()
}

func judgeBriefMarkdown(report EvaluationResult) string {
	var b strings.Builder
	b.WriteString("# Independent Judge Brief\n\n")
	b.WriteString("You are an independent evaluator for an autonomous SDLC experiment. Do not modify the generated application or experiment artifacts. Review the evidence below and write a concise critique of whether this trial satisfies the research goals, especially build quality, functional correctness, DDD discipline, and model/toolchain comparison value.\n\n")
	b.WriteString("## Trial\n\n")
	b.WriteString(fmt.Sprintf("- Trial ID: `%s`\n", report.TrialID))
	b.WriteString(fmt.Sprintf("- Tool: `%s`\n", report.Tool))
	b.WriteString(fmt.Sprintf("- Phase: `%s`\n", report.Phase))
	b.WriteString(fmt.Sprintf("- Allowed variable: `%s`\n", report.AllowedVariable))
	b.WriteString(fmt.Sprintf("- Workspace: `%s`\n", report.Workspace))
	b.WriteString(fmt.Sprintf("- Artifacts: `%s`\n\n", report.ArtifactDir))

	b.WriteString("## Rubric\n\n")
	b.WriteString("- Functional correctness: 35\n")
	b.WriteString("- DDD adherence: 20\n")
	b.WriteString("- Code quality: 15\n")
	b.WriteString("- Completeness: 15\n")
	b.WriteString("- Security: 5\n")
	b.WriteString("- Documentation: 5\n")
	b.WriteString("- Performance: 5\n\n")

	b.WriteString("## Automated Evidence\n\n")
	b.WriteString(fmt.Sprintf("- Quality gate passed: `%t`\n", report.EvaluatorPassed))
	b.WriteString(fmt.Sprintf("- Quality score: `%0.1f/100`\n", floatFromMap(report.QualityScore, "total")))
	b.WriteString(fmt.Sprintf("- Fixed verification passed: `%t`\n", report.VerificationPassed))
	b.WriteString(fmt.Sprintf("- Browser/workflow checks passed: `%t`\n", report.BrowserPassed))
	b.WriteString(fmt.Sprintf("- DDD score: `%0.0f/100`\n", floatFromMap(report.DDDScore, "score")))
	b.WriteString(fmt.Sprintf("- Completed build-plan items: `%d`\n", len(report.CompletedItems)))
	b.WriteString(fmt.Sprintf("- Tool iterations: `%d`\n", report.IterationCount))
	b.WriteString(fmt.Sprintf("- Tool runtime: `%s`\n", formatDuration(report.ToolDurationMS)))
	b.WriteString(fmt.Sprintf("- Observed tokens: `%s`\n", formatInt(report.TokenUsage.Total)))
	b.WriteString(fmt.Sprintf("- Token source: `%s`\n\n", report.TokenUsage.Source))

	b.WriteString("## Critical Findings\n\n")
	for _, finding := range criticalFindings(report) {
		b.WriteString(fmt.Sprintf("- %s\n", finding))
	}
	b.WriteString("\n")

	b.WriteString("## Required Judge Output\n\n")
	b.WriteString("- Verdict: pass, marginal, or fail for research-quality acceptance.\n")
	b.WriteString("- Top functional defects and why they matter.\n")
	b.WriteString("- DDD/domain-language critique.\n")
	b.WriteString("- Implications for RQ0, RQ2, RQ4, RQ8, RQ9, and RQ10.\n")
	b.WriteString("- Data gaps that must be closed before comparing this trial against other tools/models.\n")
	return b.String()
}

func criticalFindings(report EvaluationResult) []string {
	findings := []string{}
	if !report.VerificationPassed {
		findings = append(findings, "Fixed verification failed. Treat this as a build correctness failure, not a comparable successful trial.")
	}
	if !report.BrowserPassed {
		failures := failedBrowserRoutes(report.BrowserEvaluation)
		if len(failures) > 0 {
			findings = append(findings, fmt.Sprintf("The generated web app launches, but browser evaluation found broken or incomplete user-facing routes: %s.", strings.Join(failures, ", ")))
		} else {
			findings = append(findings, "The generated web app launches, but one or more browser workflow checks failed.")
		}
	}
	missingTerms := stringSliceFromMap(report.DDDScore, "missing_terms")
	if len(missingTerms) > 0 {
		findings = append(findings, fmt.Sprintf("DDD discipline is incomplete; missing required commercial flooring language: %s.", strings.Join(missingTerms, ", ")))
	}
	if report.BuildComplete && report.VerificationPassed && !report.BrowserPassed {
		findings = append(findings, "The build self-reported as complete and passed static verification, but independent browser testing found acceptance-surface defects. This is important evidence for RQ2, RQ4, and RQ10.")
	}
	if report.JudgeReview.Ran && report.JudgeReview.Verdict != "" && report.JudgeReview.Verdict != "pass" {
		findings = append(findings, fmt.Sprintf("The LLM judge verdict was %s; see judge-report.md for the independent critique.", report.JudgeReview.Verdict))
	}
	if !report.JudgeReview.Ran {
		findings = append(findings, "The LLM judge did not run; judge-review evidence is incomplete.")
	}
	if len(findings) == 0 {
		findings = append(findings, "No critical evaluator findings were detected by the current automated checks.")
	}
	return findings
}

func residualRisks(report EvaluationResult) []string {
	risks := []string{
		"Security scoring is provisional until static security analysis and dependency vulnerability checks are added.",
		"Cost-quality comparison has token usage but remains incomplete until provider pricing/rate capture is added.",
		"Static DDD scoring detects terminology and context signals, but it does not prove aggregate boundaries or domain event correctness.",
	}
	if report.BrowserPassed {
		risks = append(risks, "Browser checks cover critical routes and acceptance surfaces, but they are not yet full end-to-end data mutation tests.")
	} else {
		risks = append(risks, "Browser failures should be treated as app quality findings for this trial unless the evaluator route list is inconsistent with the committed spec.")
	}
	return risks
}

func failedBrowserRoutes(browserEval map[string]interface{}) []string {
	pages, ok := browserEval["pages"].([]interface{})
	if !ok {
		return []string{}
	}
	failures := []string{}
	for _, raw := range pages {
		page, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		errors := stringSliceFromMap(page, "errors")
		if len(errors) == 0 {
			continue
		}
		failures = append(failures, fmt.Sprintf("%s (%s)", fmt.Sprint(page["path"]), strings.Join(errors, "; ")))
	}
	return failures
}

func readKeyValueFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func readJSONMap(path string) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := readJSONFile(path, &out)
	return out, err
}

func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedMapKeysInt(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeysInt64(values map[string]int64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeysString(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeysStringSlice(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func archive(cfg TrialConfig) error {
	workspace := workspacePath(cfg)
	outcome := "archived"
	message := "archive command completed"
	if state, err := readBuildState(workspace); err == nil {
		switch {
		case state.Complete:
			outcome = "completed"
			message = "build state has no in-progress, blocked, or next eligible items"
		case state.Blocked:
			outcome = "blocked"
			message = state.BlockedSummary
		default:
			message = "archive command completed before build completion"
		}
	}
	return archiveWithStatus(cfg, outcome, message)
}

func archiveWithStatus(cfg TrialConfig, outcome, message string) error {
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
	finalStatus := finalStatusText(cfg, workspace, outcome, message, status)
	if err := os.WriteFile(filepath.Join(artifactDir, "final-status.txt"), []byte(finalStatus), 0o644); err != nil {
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

func finalStatusText(cfg TrialConfig, workspace, outcome, message, gitStatus string) string {
	var b strings.Builder
	b.WriteString("trial_id: ")
	b.WriteString(cfg.TrialID)
	b.WriteString("\n")
	b.WriteString("outcome: ")
	b.WriteString(outcome)
	b.WriteString("\n")
	b.WriteString("archived_at: ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\n")
	if message != "" {
		b.WriteString("message: ")
		b.WriteString(squashWhitespace(message))
		b.WriteString("\n")
	}

	if state, err := readBuildState(workspace); err == nil {
		b.WriteString("build_complete: ")
		b.WriteString(fmt.Sprintf("%t", state.Complete))
		b.WriteString("\n")
		b.WriteString("build_blocked: ")
		b.WriteString(fmt.Sprintf("%t", state.Blocked))
		b.WriteString("\n")
		b.WriteString("next_eligible: ")
		b.WriteString(state.NextEligibleSummary)
		b.WriteString("\n")
		if state.Blocked {
			b.WriteString("blocked_summary: ")
			b.WriteString(state.BlockedSummary)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("build_state_error: ")
		b.WriteString(squashWhitespace(err.Error()))
		b.WriteString("\n")
	}

	gitStatus = strings.TrimSpace(gitStatus)
	if gitStatus == "" {
		b.WriteString("workspace_git_status: clean\n")
	} else {
		b.WriteString("workspace_git_status: dirty\n")
		b.WriteString("workspace_git_status_short:\n")
		b.WriteString(gitStatus)
		b.WriteString("\n")
	}
	return b.String()
}

func squashWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func clean(cfg TrialConfig) error {
	workspace := workspacePath(cfg)
	if !exists(workspace) {
		return nil
	}
	if err := ensureWorkspacePath(cfg, workspace); err != nil {
		return err
	}
	statusf("clean: removing workspace %s", workspace)
	if err := os.RemoveAll(workspace); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}
	statusf("clean: removed workspace")
	return nil
}

func trial(cfg TrialConfig) error {
	if err := prepare(cfg); err != nil {
		return err
	}
	maxIterations := loopMaxIterations(cfg)
	for iteration := 1; iteration <= maxIterations; iteration++ {
		statusf("trial: iteration %d/%d", iteration, maxIterations)
		if err := runToolIteration(cfg, iteration); err != nil {
			_ = archiveWithStatus(cfg, "tool_failed", err.Error())
			return err
		}

		state, err := readBuildState(workspacePath(cfg))
		if err != nil {
			_ = archiveWithStatus(cfg, "build_state_unreadable", err.Error())
			return err
		}
		if state.Blocked {
			_ = archiveWithStatus(cfg, "blocked", state.BlockedSummary)
			return fmt.Errorf("build blocked after iteration %d: %s", iteration, state.BlockedSummary)
		}

		if err := verifyIteration(cfg, iteration); err != nil {
			_ = archiveWithStatus(cfg, "verification_failed", err.Error())
			return err
		}
		if state.Complete {
			if err := archiveWithStatus(cfg, "completed", fmt.Sprintf("build complete after %d iterations", iteration)); err != nil {
				return err
			}
			statusf("trial: build complete after %d iterations", iteration)
			statusf("trial: workspace retained for manual testing: %s", workspacePath(cfg))
			statusf("trial: run clean explicitly when testing is complete")
			return nil
		}
		statusf("trial: continuing; next eligible: %s", state.NextEligibleSummary)
	}
	if err := archiveWithStatus(cfg, "max_iterations_reached", fmt.Sprintf("max_iterations reached before build completion: %d", maxIterations)); err != nil {
		return err
	}
	statusf("trial: workspace retained for manual testing: %s", workspacePath(cfg))
	statusf("trial: run clean explicitly when testing is complete")
	return fmt.Errorf("max_iterations reached before build completion: %d", maxIterations)
}

type BuildStateStatus struct {
	Blocked             bool
	Complete            bool
	BlockedSummary      string
	NextEligibleSummary string
}

func readBuildState(workspace string) (BuildStateStatus, error) {
	path := filepath.Join(workspace, "BUILD_STATE.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return BuildStateStatus{}, fmt.Errorf("read BUILD_STATE.md: %w", err)
	}
	content := string(b)
	status := strings.ToLower(strings.TrimSpace(firstHeaderValue(content, "## Status:")))
	blockedItems := sectionItems(content, "## Blocked")
	inProgressItems := sectionItems(content, "## In Progress")
	nextEligibleItems := sectionItems(content, "## Next Eligible")
	completedItems := sectionItems(content, "## Completed")

	blocked := strings.Contains(status, "blocked") || len(blockedItems) > 0
	blockedSummary := "status=blocked"
	if len(blockedItems) > 0 {
		blockedSummary = strings.Join(blockedItems, "; ")
	}
	nextSummary := "none"
	if len(nextEligibleItems) > 0 {
		nextSummary = strings.Join(nextEligibleItems, ", ")
	}
	complete := !blocked && len(inProgressItems) == 0 && len(nextEligibleItems) == 0 && len(completedItems) > 0

	return BuildStateStatus{
		Blocked:             blocked,
		Complete:            complete,
		BlockedSummary:      blockedSummary,
		NextEligibleSummary: nextSummary,
	}, nil
}

func firstHeaderValue(content, prefix string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func sectionItems(content, header string) []string {
	lines := strings.Split(content, "\n")
	inSection := false
	var items []string
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			inSection = isSectionHeader(line, header)
			continue
		}
		if !inSection || line == "" {
			continue
		}
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") {
			item := strings.TrimSpace(line[5:])
			if !isEmptySectionItem(item) {
				items = append(items, item)
			}
			continue
		}
		if strings.HasPrefix(line, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if !isEmptySectionItem(item) {
				items = append(items, item)
			}
		}
	}
	return items
}

func isSectionHeader(line, header string) bool {
	if line == header {
		return true
	}
	return strings.HasPrefix(line, header+" ")
}

func isEmptySectionItem(item string) bool {
	switch strings.ToLower(strings.TrimSpace(item)) {
	case "", "none", "(none)", "none yet.", "(none yet)":
		return true
	default:
		return false
	}
}

func loopMaxIterations(cfg TrialConfig) int {
	const defaultMaxIterations = 1
	raw, ok := cfg.Loop["max_iterations"]
	if !ok {
		return defaultMaxIterations
	}
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultMaxIterations
}

func iterationFile(prefix string, iteration int, suffix string) string {
	if iteration <= 0 {
		return prefix + "-" + suffix
	}
	return fmt.Sprintf("%s-iteration-%03d-%s", prefix, iteration, suffix)
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

func trialEnv(cfg TrialConfig, workspace, artifactDir string) (map[string]string, error) {
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
		"TURBO_TELEMETRY_DISABLED":     "1",
		"TURBO_NO_UPDATE_NOTIFIER":     "1",
	}
	if certFile := hostCertFile(); certFile != "" {
		env["SSL_CERT_FILE"] = certFile
	}
	if cfg.Tool.Name == "codex" {
		codexHome, err := prepareCodexHome(cacheRoot)
		if err != nil {
			return nil, err
		}
		env["CODEX_HOME"] = codexHome
	}
	for k, v := range cfg.Tool.Env {
		env[k] = v
	}
	return env, nil
}

func hostCertFile() string {
	for _, path := range []string{
		"/opt/homebrew/etc/ca-certificates/cert.pem",
		"/opt/homebrew/etc/openssl@3/cert.pem",
		"/etc/ssl/cert.pem",
		"/private/etc/ssl/cert.pem",
	} {
		if exists(path) {
			return path
		}
	}
	return ""
}

func prepareCodexHome(cacheRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sourceHome := filepath.Join(home, ".codex")
	destHome := filepath.Join(cacheRoot, "codex-home")
	if err := os.MkdirAll(destHome, 0o700); err != nil {
		return "", err
	}
	for _, name := range []string{"config.toml", "auth.json"} {
		src := filepath.Join(sourceHome, name)
		if !exists(src) {
			continue
		}
		dst := filepath.Join(destHome, name)
		if err := copyFile(src, dst, 0o600); err != nil {
			return "", fmt.Errorf("copy Codex %s: %w", name, err)
		}
	}
	return destHome, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
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
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() || info.Mode()&os.ModeSymlink != 0 {
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
	for _, part := range parts {
		switch part {
		case ".git", "node_modules", ".next", "dist", "build", ".turbo", "coverage":
			return true
		}
	}
	return false
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

func ensureWorkspacePath(cfg TrialConfig, workspace string) error {
	workspaceRoot, err := filepath.Abs(cfg.WorkspaceRoot)
	if err != nil {
		return err
	}
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	expected := filepath.Join(workspaceRoot, cfg.TrialID)
	if workspaceAbs != expected {
		return fmt.Errorf("refusing to clean unexpected workspace path: %s", workspaceAbs)
	}
	rel, err := filepath.Rel(workspaceRoot, workspaceAbs)
	if err != nil {
		return err
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing to clean workspace outside workspace root: %s", workspaceAbs)
	}
	return nil
}

func exitOnErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
