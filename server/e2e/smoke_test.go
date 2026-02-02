//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const repoRootRel = ".."   // relative to ./e2e
const mainPkgRel = "./cmd" // main.go lives in cmd/

func TestSmoke_Healthz(t *testing.T) {
	repoRoot := repoRootPath(t)

	// Start SQLite "service" container that creates a DB file in a host temp dir
	sqlitePath := startSQLite(t)

	bin := buildBinary(t, repoRoot)
	addr := pickFreeAddr(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"APP_ENV=dev",
		"LOG_LEVEL=info",
		"HTTP_ADDR="+addr,

		// DB envs (match your db package)
		"DB_DRIVER=sqlite3",
		"SQLITE_PATH="+sqlitePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://" + addr + "/healthz"

	waitForOK(t, client, url, 5*time.Second)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body.status=%q want=%q", body["status"], "ok")
	}

	stopServer(t, cmd)
}

func startSQLite(t *testing.T) string {
	t.Helper()

	hostDir := t.TempDir()
	dbPath := filepath.Join(hostDir, "app.db")

	// Optional: pre-create the file (not required for SQLite, but harmless)
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create sqlite db file: %v", err)
	}
	_ = f.Close()

	return dbPath
}

func repoRootPath(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	repo := filepath.Clean(filepath.Join(wd, repoRootRel))
	if _, err := os.Stat(filepath.Join(repo, "go.mod")); err != nil {
		t.Fatalf("repo root %q does not contain go.mod: %v", repo, err)
	}

	return repo
}

func buildBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	tmp := t.TempDir()
	out := filepath.Join(tmp, "cloudpico-server")

	build := exec.Command("go", "build", "-o", out, mainPkgRel)
	build.Dir = repoRoot
	build.Env = os.Environ()

	b, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(b))
	}

	return out
}

func pickFreeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen :0: %v", err)
	}
	defer ln.Close()

	return ln.Addr().String()
}

func waitForOK(t *testing.T, client *http.Client, url string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server not healthy after %s: %s", timeout, url)
}

func stopServer(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	_ = cmd.Process.Signal(syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		t.Fatalf("server did not exit in time")
	case err := <-done:
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("server exited non-zero: %v", err)
			}
			t.Fatalf("server wait error: %v", err)
		}
	}
}
