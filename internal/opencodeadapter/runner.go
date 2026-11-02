package opencodeadapter

// Owned-process lifecycle: spawn one private `opencode acp` child in its own
// process group and terminate only that group. No shared service, login, fork,
// child session, arbitrary argv or inherited HOME/XDG/provider/proxy environment.

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// processStream bundles the owned process's stdin (write) and stdout (read)
// pipes into one deadline-capable stream. stderr is discarded: ACP runs on
// stdout, and `--print-logs` is never passed.
type processStream struct {
	stdin  *os.File
	stdout *os.File
}

func (p *processStream) Read(b []byte) (int, error)  { return p.stdout.Read(b) }
func (p *processStream) Write(b []byte) (int, error) { return p.stdin.Write(b) }
func (p *processStream) Close() error {
	_ = p.stdin.Close()
	return p.stdout.Close()
}
func (p *processStream) SetReadDeadline(t time.Time) error  { return p.stdout.SetReadDeadline(t) }
func (p *processStream) SetWriteDeadline(t time.Time) error { return p.stdin.SetWriteDeadline(t) }

// process is one owned ACP child. It is always started in its own process group
// so termination can target only it and its descendants.
type process struct {
	cmd    *exec.Cmd
	stream *processStream
	waitCh chan error
}

// spawnOwnedProcess starts the fixed executable with the given arguments in a
// fresh process group, with an explicitly allowlisted (not inherited)
// environment and no credential-bearing variables. stdout carries ACP; stderr
// is discarded.
func spawnOwnedProcess(executable string, args, env []string) (*process, error) {
	return spawnProcess(executable, args, env, 1)
}

// spawnProcess is the testable core. acpFD selects which child descriptor
// carries ACP output: fd 1 for the production `opencode acp` child, fd 3 (via
// ExtraFiles) for a fake peer whose fd 1 belongs to the test runner.
func spawnProcess(executable string, args, env []string, acpFD int) (*process, error) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create stdin pipe")
	}
	acpR, acpW, err := os.Pipe()
	if err != nil {
		stdinR.Close()
		stdinW.Close()
		return nil, fmt.Errorf("cannot create stdout pipe")
	}
	cmd := exec.Command(executable, args...)
	cmd.Stdin = stdinR
	cmd.Stderr = io.Discard
	if acpFD == 1 {
		cmd.Stdout = acpW
	} else {
		cmd.Stdout = io.Discard
		cmd.ExtraFiles = []*os.File{acpW}
	}
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		stdinR.Close()
		stdinW.Close()
		acpR.Close()
		acpW.Close()
		return nil, fmt.Errorf("cannot start owned process")
	}
	stdinR.Close()
	acpW.Close()
	p := &process{
		cmd:    cmd,
		stream: &processStream{stdin: stdinW, stdout: acpR},
		waitCh: make(chan error, 1),
	}
	go func() { p.waitCh <- cmd.Wait() }()
	return p, nil
}

// Stream returns the owned process's ACP byte stream.
func (p *process) Stream() acpStream { return p.stream }

// Terminate closes the transport and signals only the owned process group. It
// never signals the parent or an unrelated process group.
func (p *process) Terminate() error {
	_ = p.stream.stdin.Close()
	pgid := p.cmd.Process.Pid
	if p.cmd.SysProcAttr != nil && p.cmd.SysProcAttr.Setpgid {
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
			return fmt.Errorf("cannot signal owned process group")
		}
	}
	return nil
}

// Wait reaps the owned process. It is not a success signal: a provider failure
// can still exit zero, so callers never treat Wait alone as task success.
func (p *process) Wait() error {
	select {
	case err := <-p.waitCh:
		return err
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		<-p.waitCh
		return fmt.Errorf("owned process did not exit promptly")
	}
}
