package wallet

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessLock(t *testing.T) {
	if os.Getenv("KIKO_LOCK_CHILD") == "1" {
		runLockChild(t)
		return
	}

	dir := t.TempDir()
	s := &Store{
		ConfigDir: filepath.Join(dir, "config"),
		DataDir:   filepath.Join(dir, "data"),
	}
	if err := s.ensureDirs(); err != nil {
		t.Fatal(err)
	}
	held, err := lockFile(s.coinLockPath("cardano"))
	if err != nil {
		t.Fatal(err)
	}
	released := false
	defer func() {
		if !released {
			unlockFile(held)
		}
	}()

	ready := filepath.Join(dir, "ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessLock$")
	cmd.Env = append(os.Environ(),
		"KIKO_LOCK_CHILD=1",
		"KIKO_CONFIG="+s.ConfigDir,
		"KIKO_DATA="+s.DataDir,
		"KIKO_READY="+ready,
	)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case err := <-waitErr:
			t.Fatalf("child exited before the lock: %v\n%s", err, output.String())
		default:
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("child did not reach the lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("child exited instead of blocking: %v\n%s", err, output.String())
	}

	const address = "addr1parent"
	if err := s.writePublic("cardano", Public{Address: address}); err != nil {
		_ = cmd.Process.Kill()
		t.Fatal(err)
	}
	unlockFile(held)
	released = true

	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("child: %v\n%s", err, output.String())
		}
		if !strings.Contains(output.String(), "ADDRESS:"+address) {
			t.Fatalf("child output:\n%s", output.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child did not finish after the lock was released")
	}
}

func runLockChild(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(os.Getenv("KIKO_READY"), []byte("1"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{ConfigDir: os.Getenv("KIKO_CONFIG"), DataDir: os.Getenv("KIKO_DATA")}
	err := s.withCoin("cardano", func() error {
		pub, err := s.readPublic("cardano")
		if err != nil {
			return err
		}
		fmt.Println("ADDRESS:" + pub.Address)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
