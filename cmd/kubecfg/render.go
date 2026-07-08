package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/amimof/kubecfg/pkg/cmdutil"
	"github.com/amimof/kubecfg/pkg/command"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/amimof/kubecfg/pkg/service"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	fzf "github.com/junegunn/fzf/src"
)

var (
	errNoSelection = errors.New("no selection")
	fzfRun         = fzf.Run
)

func newRenderCmd() *cobra.Command {
	var (
		noLogin     bool
		noUse       bool
		all         bool
		waitTimeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "render [WORKSPACE] [KUBECONFIG]",
		Short: "Select and render kubeconfigs",
		Long:  `Select a kubeconfig, render and write it to the base directory.`,
		Example: `
# Pick kubeconfig from fuzzy finder
kubecfg render

# Render kubeconfig mainframe in workspace homelab
kubecfg render homelab mainframe

# Render all kubeconfigs across all workspaces
kubecfg render --all`,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			if all {
				return runRenderAll(cmd.Context(), noLogin, waitTimeout)
			}
			switch len(args) {
			case 0:
				return runRenderCmdFzf(cmd.Context(), noLogin, waitTimeout)
			case 1:
				return runRenderCmd(cmd.Context(), args[0], "", noLogin, waitTimeout)
			default:
				return runRenderCmd(cmd.Context(), args[0], args[1], noLogin, waitTimeout)
			}
		}),
	}

	cmd.PersistentFlags().BoolVar(&noLogin, "no-login", false, "Skip execution of login flow prior to kubeconfig rendering")
	cmd.PersistentFlags().BoolVar(&noUse, "no-use", false, "Skip activation of rendered kubeconfig after successful render")
	cmd.PersistentFlags().BoolVarP(&all, "all", "a", false, "Render all kubeconfigs across all workspaces")
	cmd.PersistentFlags().DurationVar(&waitTimeout, "timeout", time.Second*30, "How long in seconds to wait for login opearation to finish before giving up")

	return cmd
}

func writeKubeconfig(path string, kubeconfig *api.Config) error {
	data, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// setConfig creates a new symlink to a kubeconfigfile overwriting any existing one
func setConfig(baseDir, name string) error {
	dst := path.Join(baseDir, "config")
	// Remove existing symlink to config so we don't run into an error
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		err := os.Remove(dst)
		if err != nil {
			return err
		}
	}
	// Create the symlink to config
	return os.Symlink(name, path.Join(baseDir, "config"))
}

// renderTask describes a single kubeconfig to render, along with its display name.
type renderTask struct {
	displayName string // "workspace/kubeconfig"
	rk          *config.RuntimeKubeconfig
}

// renderKubeconfigs renders a list of kubeconfigs concurrently, showing a dashboard
// with a spinner per kubeconfig. Errors on individual kubeconfigs do not block
// others. Returns a joined error if any kubeconfigs failed, nil otherwise.
func renderKubeconfigs(ctx context.Context, tasks []renderTask, skipLogin bool, waitTimeout time.Duration) error {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = t.displayName
	}

	dash, err := cmdutil.NewDashboard(names, cmdutil.WithLayout(&cmdutil.Layout{Padding: [4]int{0, 2, 0, 0}}), cmdutil.WithFields(cmdutil.FieldError))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go dash.Loop(ctx)

	var (
		outerWg      sync.WaitGroup
		mu           sync.Mutex
		renderErrors []error
	)

	for i, task := range tasks {
		outerWg.Add(1)
		go func(idx int, t renderTask) {
			defer outerWg.Done()

			if err := renderSingleKubeconfig(ctx, t.rk, skipLogin, waitTimeout); err != nil {
				dash.FailMsg(idx, err.Error())
				mu.Lock()
				renderErrors = append(renderErrors, fmt.Errorf("%s: %w", t.displayName, err))
				mu.Unlock()
				return
			}

			dash.DoneMsg(idx, t.displayName)
		}(i, task)
	}

	outerWg.Wait()
	cancel()

	// Wait for the dashboard render loop to finish (final frame)
	dash.Wait()

	if len(renderErrors) > 0 {

		errHeader := fmt.Sprintf("\n%d of %d kubeconfigs failed to render\n", len(renderErrors), len(tasks))
		cmdutil.Printf("{{ .Error }}", cmdutil.Data{"Error": errHeader})
		return fmt.Errorf("%w", errors.Join(renderErrors...))
	}

	return nil
}

// renderSingleKubeconfig runs login sources, applies imports, and writes the
// kubeconfig file for a single RuntimeKubeconfig. All login sources within the
// kubeconfig are executed concurrently.
func renderSingleKubeconfig(ctx context.Context, rk *config.RuntimeKubeconfig, skipLogin bool, waitTimeout time.Duration) error {
	if !skipLogin {
		var (
			loginWg  sync.WaitGroup
			loginMu  sync.Mutex
			loginErr error
		)

		for _, source := range rk.LoginSources {
			loginWg.Add(1)
			go func(s *config.RuntimeLoginSource) {
				defer loginWg.Done()
				stdout := &bytes.Buffer{}
				stderr := &bytes.Buffer{}
				if err := runLogin(ctx, s, waitTimeout, stdout, stderr); err != nil {
					loginMu.Lock()
					loginErr = err
					loginMu.Unlock()
				}
			}(source)
		}

		loginWg.Wait()

		if loginErr != nil {
			return loginErr
		}
	}

	if err := applyImportedContexts(rk); err != nil {
		return err
	}

	if err := writeKubeconfig(rk.Path, rk.Config); err != nil {
		return err
	}

	return nil
}

func runRenderCmd(ctx context.Context, workspaceName, kubeconfigName string, skipLogin bool, waitTimeout time.Duration) error {
	if workspaceName == "" {
		return fmt.Errorf("workspace cannot be empty")
	}

	compiler, err := newCompilerWithOptionalDecryptor(&cfg, cfg.IdentityFiles)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	if !runtime.WorkspaceExists(workspaceName) {
		return fmt.Errorf("workspace does not exist: %s", workspaceName)
	}

	var kubeconfigs []*config.RuntimeKubeconfig

	if kubeconfigName != "" {
		if !runtime.KubeconfigExists(workspaceName, kubeconfigName) {
			return fmt.Errorf("kubeconfig does not exist: %s/%s", workspaceName, kubeconfigName)
		}
		rk := runtime.Workspace(workspaceName).Kubeconfig(kubeconfigName)
		kubeconfigs = append(kubeconfigs, rk)
	} else {
		for _, rtkc := range runtime.Workspace(workspaceName).Kubeconfigs {
			kubeconfigs = append(kubeconfigs, rtkc)
		}
	}

	tasks := make([]renderTask, len(kubeconfigs))
	for i, rk := range kubeconfigs {
		tasks[i] = renderTask{
			displayName: fmt.Sprintf("%s/%s", workspaceName, rk.Name),
			rk:          rk,
		}
	}

	cmdutil.Println("Rendering workspace\n")

	if err := renderKubeconfigs(ctx, tasks, skipLogin, waitTimeout); err != nil {
		return err
	}

	// Automatically run "use" when only 1 kubeconfig
	if len(kubeconfigs) == 1 {
		rk := kubeconfigs[0]
		if err := setConfig(runtime.BaseDir, rk.Path); err != nil {
			return err
		}
		fmt.Print("\n")
		cmdutil.Printf(`{{ "✔" | FgGreen }} Using kubeconfig {{ .Workspace | FgYellow }}/{{ .Kubeconfig | FgCyan }}`, cmdutil.Data{"Workspace": workspaceName, "Kubeconfig": rk.Name})
	}

	return nil
}

func runRenderAll(ctx context.Context, skipLogin bool, waitTimeout time.Duration) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, cfg.IdentityFiles)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	var tasks []renderTask
	for _, ws := range runtime.Workspaces {
		for _, rk := range ws.Kubeconfigs {
			tasks = append(tasks, renderTask{
				displayName: fmt.Sprintf("%s/%s", ws.Name, rk.Name),
				rk:          rk,
			})
		}
	}

	if len(tasks) == 0 {
		return fmt.Errorf("no kubeconfigs found")
	}

	cmdutil.Println("Rendering all workspaces\n")

	return renderKubeconfigs(ctx, tasks, skipLogin, waitTimeout)
}

func runRenderCmdFzf(ctx context.Context, skipLogin bool, waitTimeout time.Duration) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, cfg.IdentityFiles)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	workspace, selected, err := pickContext(runtime)
	if err != nil {
		if errors.Is(err, errNoSelection) {
			return nil
		}
		return err
	}

	rk := runtime.Workspace(workspace).Kubeconfig(selected)

	if rk.Config.CurrentContext == "" {
		rk.Config.CurrentContext = rk.Name
	}

	tasks := []renderTask{{
		displayName: fmt.Sprintf("%s/%s", workspace, selected),
		rk:          rk,
	}}

	cmdutil.Println("Rendering kubeconfig\n")

	if err := renderKubeconfigs(ctx, tasks, skipLogin, waitTimeout); err != nil {
		return err
	}

	if err := setConfig(runtime.BaseDir, rk.Path); err != nil {
		return err
	}
	cmdutil.Printf(`{{ "✔" | FgGreen }} Using kubeconfig {{ .Workspace | FgYellow }}/{{ .Kubeconfig | FgCyan }}`, cmdutil.Data{"Workspace": workspace, "Kubeconfig": selected})

	return nil
}

func runLogin(ctx context.Context, source *config.RuntimeLoginSource, waitTimeout time.Duration, stdout, stderr *bytes.Buffer) error {
	cmdCtx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()

	runner := command.NewExecCommandRunner()
	loginService := service.LoginService{Runner: runner, Stdout: stdout, Stderr: stderr}

	err := loginService.Login(cmdCtx, source)
	if err != nil {
		return err
	}

	return nil
}

func compactLoginErrorDetail(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}

	lines := strings.FieldsFunc(stderr, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	if len(lines) == 0 {
		return ""
	}

	return strings.TrimSpace(lines[len(lines)-1])
}

func pickContext(rc *config.RuntimeConfig) (string, string, error) {
	inputChan := make(chan string)
	go func() {
		for _, w := range rc.Workspaces {
			for _, k := range w.Kubeconfigs {
				input := fmt.Sprintf("%s/%s", w.Name, k.Name)
				inputChan <- input

			}
		}
		close(inputChan)
	}()

	selected, err := pick(inputChan)
	if err != nil {
		return "", "", err
	}

	var workspaceName string
	ss := strings.Split(selected, "/")
	if len(ss) == 2 {
		workspaceName = ss[0]
		return ss[0], ss[1], nil
	}

	return workspaceName, selected, nil
}

func pick(input chan string) (string, error) {
	outputChan := make(chan string, 1)

	// Build fzf.Options
	options, err := fzf.ParseOptions(
		true,
		[]string{"--reverse", "--border", "--height=40%"},
	)
	if err != nil {
		return "", fmt.Errorf("fzf exit error %d: %w", fzf.ExitError, err)
	}

	// Set up input and output channels
	options.Input = input
	options.Output = outputChan

	// Run fzf
	code, err := fzfRun(options)
	if err != nil {
		return "", fmt.Errorf("fzf exited with code %d: %w", code, err)
	}

	switch code {
	case fzf.ExitInterrupt, fzf.ExitNoMatch:
		return "", errNoSelection
	case fzf.ExitOk:
	default:
		return "", fmt.Errorf("fzf exited with code %d", code)
	}

	var selected string
	select {
	case selected = <-outputChan:
	default:
		return "", fmt.Errorf("fzf exited with code %d without a selection", code)
	}

	return selected, nil
}
