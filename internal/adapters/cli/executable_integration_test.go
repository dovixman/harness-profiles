package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var builtHP struct {
	once sync.Once
	path string
	err  error
}

type hpResult struct {
	stdout string
	stderr string
}

func TestExecutableAddMissingRootCreatesProfileAndSymlink(t *testing.T) {
	env := newExecutableEnv(t)
	root := filepath.Join(env.workDir, "missing-root")

	env.run(t, "add", "--label", "Claude", "--link", root, "--profile", "default", "claude")

	profile := filepath.Join(env.configHome, "harnesses", "claude", "default", "root")
	assertDir(t, profile)
	assertSymlinkTarget(t, root, profile)
	result := env.run(t, "claude", "current")
	if strings.TrimSpace(result.stdout) != "default" {
		t.Fatalf("current stdout = %q, want default", result.stdout)
	}
}

func TestExecutableMultiLinkDirAndFileLifecycle(t *testing.T) {
	env := newExecutableEnv(t)
	runtime := filepath.Join(env.workDir, "runtime")
	state := filepath.Join(env.workDir, "runtime.json")
	mustWriteExecutableFixture(t, filepath.Join(runtime, "settings.json"), "runtime")
	mustWriteExecutableFixture(t, state, "state")

	env.run(t, "add", "--label", "Claude", "--link", "root="+runtime, "--link", "state="+state, "--profile", "default", "claude")
	assertSymlinkTarget(t, runtime, filepath.Join(env.configHome, "harnesses", "claude", "default", "root"))
	assertSymlinkTarget(t, state, filepath.Join(env.configHome, "harnesses", "claude", "default", "state"))

	env.run(t, "claude", "clone", "default", "work")
	env.run(t, "claude", "switch", "work")
	assertSymlinkTarget(t, runtime, filepath.Join(env.configHome, "harnesses", "claude", "work", "root"))
	assertSymlinkTarget(t, state, filepath.Join(env.configHome, "harnesses", "claude", "work", "state"))

	env.run(t, "delete", "--mode", "restore", "--profile", "work", "claude")
	assertRegularFile(t, filepath.Join(runtime, "settings.json"))
	assertRegularFile(t, state)
}

func TestExecutableHappyPathProfileLifecycle(t *testing.T) {
	env := newExecutableEnv(t)
	root := filepath.Join(env.workDir, "runtime")
	mustWriteExecutableFixture(t, filepath.Join(root, "settings.json"), "default")
	env.run(t, "add", "--label", "Claude", "--link", root, "--profile", "default", "claude")

	env.run(t, "claude", "clone", "default", "work")
	env.run(t, "claude", "switch", "work")
	current := env.run(t, "claude", "current")
	if strings.TrimSpace(current.stdout) != "work" {
		t.Fatalf("current stdout = %q, want work", current.stdout)
	}
	activeDelete := env.runFail(t, "claude", "delete", "--yes", "work")
	if !strings.Contains(activeDelete.stderr, "cannot delete active profile") {
		t.Fatalf("active delete stderr = %q, want active profile refusal", activeDelete.stderr)
	}
	env.run(t, "claude", "switch", "default")
	env.run(t, "claude", "delete", "--yes", "work")

	list := env.run(t, "claude", "ls")
	if strings.Contains(list.stdout, "work") || !strings.Contains(list.stdout, "default") {
		t.Fatalf("profile list stdout = %q, want only default", list.stdout)
	}
}

func TestExecutableUpdateHarnessRepointsRootAndDeleteModes(t *testing.T) {
	t.Run("update repoints and removes old symlink", func(t *testing.T) {
		env := newExecutableEnv(t)
		oldRoot := filepath.Join(env.workDir, "old-root")
		env.run(t, "add", "--link", oldRoot, "--profile", "default", "claude")
		newRoot := filepath.Join(env.workDir, "new-root")

		env.run(t, "update", "--label", "Claude Updated", "--link", newRoot, "--remove-old", "claude")

		profile := filepath.Join(env.configHome, "harnesses", "claude", "default", "root")
		assertSymlinkTarget(t, newRoot, profile)
		assertMissing(t, oldRoot)
		list := env.run(t, "ls")
		if !strings.Contains(list.stdout, "Claude Updated") || !strings.Contains(list.stdout, newRoot) {
			t.Fatalf("ls stdout = %q, want updated label and root", list.stdout)
		}
	})

	t.Run("delete restore replaces symlink with profile contents", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		mustWriteExecutableFixture(t, filepath.Join(root, "settings.json"), "restored")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")

		env.run(t, "delete", "--mode", "restore", "--profile", "default", "claude")

		assertRegularFile(t, filepath.Join(root, "settings.json"))
		assertMissing(t, filepath.Join(env.configHome, "harnesses", "claude"))
		list := env.run(t, "ls")
		if strings.Contains(list.stdout, "claude") {
			t.Fatalf("ls stdout = %q, want harness removed", list.stdout)
		}
	})

	t.Run("delete all requires confirmation and deletes root", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")

		failed := env.runFail(t, "delete", "--mode", "delete-all", "claude")
		if !strings.Contains(failed.stderr, "delete-all requires --confirm claude") {
			t.Fatalf("delete-all stderr = %q, want confirmation refusal", failed.stderr)
		}
		env.run(t, "delete", "--mode", "delete-all", "--confirm", "claude", "claude")

		assertMissing(t, root)
		assertMissing(t, filepath.Join(env.configHome, "harnesses", "claude"))
	})
}

func TestExecutableHardensAgainstManualFilesystemChanges(t *testing.T) {
	t.Run("deleted root path can be recreated by switching managed profile", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")
		if err := os.Remove(root); err != nil {
			t.Fatalf("Remove() error = %v", err)
		}

		env.run(t, "claude", "switch", "default")

		assertSymlinkTarget(t, root, filepath.Join(env.configHome, "harnesses", "claude", "default", "root"))
	})

	t.Run("deleted profile folder refuses switch without touching current root", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")
		env.run(t, "claude", "clone", "default", "work")
		work := filepath.Join(env.configHome, "harnesses", "claude", "work")
		if err := os.RemoveAll(work); err != nil {
			t.Fatalf("RemoveAll() error = %v", err)
		}

		failed := env.runFail(t, "claude", "switch", "work")

		if !strings.Contains(failed.stderr, "no such file") {
			t.Fatalf("switch stderr = %q, want missing profile error", failed.stderr)
		}
		assertSymlinkTarget(t, root, filepath.Join(env.configHome, "harnesses", "claude", "default", "root"))
	})

	t.Run("deleted active profile remains reported as current but cannot be deleted", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")
		profile := filepath.Join(env.configHome, "harnesses", "claude", "default", "root")
		if err := os.RemoveAll(profile); err != nil {
			t.Fatalf("RemoveAll() error = %v", err)
		}

		current := env.run(t, "claude", "current")
		if strings.TrimSpace(current.stdout) != "" {
			t.Fatalf("current stdout = %q, want empty after profile removal", current.stdout)
		}
		failed := env.runFail(t, "claude", "delete", "--yes", "default")
		if !strings.Contains(failed.stderr, "repair or switch first") {
			t.Fatalf("delete stderr = %q, want repair/switch refusal", failed.stderr)
		}
	})

	t.Run("external symlink is reported and can be repaired by switch", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")
		external := filepath.Join(env.workDir, "external")
		mustWriteExecutableFixture(t, filepath.Join(external, "settings.json"), "external")
		if err := os.Remove(root); err != nil {
			t.Fatalf("Remove() error = %v", err)
		}
		if err := os.Symlink(external, root); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}

		current := env.run(t, "claude", "current")
		if !strings.Contains(current.stdout, "external") || !strings.Contains(current.stdout, external) {
			t.Fatalf("current stdout = %q, want external symlink report", current.stdout)
		}
		failed := env.runFail(t, "update", "--link", filepath.Join(env.workDir, "new-root"), "claude")
		if !strings.Contains(failed.stderr, "current profile is not managed") {
			t.Fatalf("update stderr = %q, want unmanaged current refusal", failed.stderr)
		}

		env.run(t, "claude", "switch", "default")
		assertSymlinkTarget(t, root, filepath.Join(env.configHome, "harnesses", "claude", "default", "root"))
	})

	t.Run("real directory at root path refuses switch and can be adopted", func(t *testing.T) {
		env := newExecutableEnv(t)
		root := filepath.Join(env.workDir, "runtime")
		env.run(t, "add", "--link", root, "--profile", "default", "claude")
		if err := os.Remove(root); err != nil {
			t.Fatalf("Remove() error = %v", err)
		}
		mustWriteExecutableFixture(t, filepath.Join(root, "manual.json"), "manual")

		failed := env.runFail(t, "claude", "switch", "default")
		if !strings.Contains(failed.stderr, "exists and is not a symlink") {
			t.Fatalf("switch stderr = %q, want real-directory refusal", failed.stderr)
		}
		env.run(t, "claude", "adopt", "manual")

		manualProfile := filepath.Join(env.configHome, "harnesses", "claude", "manual", "root")
		assertSymlinkTarget(t, root, manualProfile)
		assertRegularFile(t, filepath.Join(manualProfile, "manual.json"))
	})
}

type executableEnv struct {
	bin        string
	configHome string
	workDir    string
}

func newExecutableEnv(t *testing.T) executableEnv {
	t.Helper()
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return executableEnv{
		bin:        hpBinary(t),
		configHome: filepath.Join(base, "config-home"),
		workDir:    workDir,
	}
}

func hpBinary(t *testing.T) string {
	t.Helper()
	builtHP.once.Do(func() {
		root, err := repoRoot()
		if err != nil {
			builtHP.err = err
			return
		}
		dir, err := os.MkdirTemp("", "hp-executable-test-*")
		if err != nil {
			builtHP.err = err
			return
		}
		builtHP.path = filepath.Join(dir, "hp")
		cmd := exec.Command("go", "build", "-o", builtHP.path, "./cmd/hp")
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			builtHP.err = &commandError{err: err, output: string(output)}
		}
	})
	if builtHP.err != nil {
		t.Fatalf("build hp executable: %v", builtHP.err)
	}
	return builtHP.path
}

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func (e executableEnv) run(t *testing.T, args ...string) hpResult {
	t.Helper()
	result, err := e.runCommand(args...)
	if err != nil {
		t.Fatalf("hp %s failed: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, result.stdout, result.stderr)
	}
	return result
}

func (e executableEnv) runFail(t *testing.T, args ...string) hpResult {
	t.Helper()
	result, err := e.runCommand(args...)
	if err == nil {
		t.Fatalf("hp %s succeeded unexpectedly\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.stdout, result.stderr)
	}
	return result
}

func (e executableEnv) runCommand(args ...string) (hpResult, error) {
	cmd := exec.Command(e.bin, args...)
	cmd.Env = append(os.Environ(), "HP_CONFIG_HOME="+e.configHome)
	cmd.Dir = e.workDir
	stdout, stderr := strings.Builder{}, strings.Builder{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return hpResult{stdout: stdout.String(), stderr: stderr.String()}, err
}

type commandError struct {
	err    error
	output string
}

func (e *commandError) Error() string {
	return e.err.Error() + "\n" + e.output
}

func mustWriteExecutableFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertSymlinkTarget(t *testing.T, link, target string) {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink(%q) error = %v", link, err)
	}
	if got != target {
		t.Fatalf("Readlink(%q) = %q, want %q", link, got, target)
	}
}

func assertDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}

func assertRegularFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v", path, err)
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%q mode = %v, want regular file", path, info.Mode())
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", path, err)
	}
}
