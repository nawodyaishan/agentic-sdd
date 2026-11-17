//go:build linux

package execution

// Linux pinned completion. The leader's exit is observed with waitid(WNOWAIT),
// which leaves the zombie unreaped, so its pid and the owned group id stay
// pinned. The owned group is then inspected through /proc, settled, and only
// after that is the leader reaped. No descriptor is kept open for the group:
// /proc reads and the waitid call use no descriptor that a child could inherit,
// and Go opens every file with O_CLOEXEC, which exec then honours.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// waitidPidType is P_PID for waitid.
const waitidPidType = 1

// observeLeaderExit blocks until the leader process exits and does not reap it.
// It returns an error only when the kernel cannot report the leader's state.
func observeLeaderExit(pid int) error {
	var info [128]byte // siginfo_t; the status is read later through cmd.Wait
	for {
		_, _, errno := syscall.Syscall6(syscall.SYS_WAITID, waitidPidType, uintptr(pid),
			uintptr(unsafe.Pointer(&info[0])), uintptr(syscall.WEXITED|syscall.WNOWAIT), 0, 0)
		if errno == 0 {
			return nil
		}
		if errno == syscall.EINTR {
			continue
		}
		return fmt.Errorf("leader exit is unobservable: %w", errno)
	}
}

// listCheckGroupMembers enumerates the owned group through /proc. A process that
// vanishes mid-scan is skipped; any other unreadable or malformed entry is
// uncertainty and fails the observation.
func listCheckGroupMembers(pgid, leader int) (checkGroupMembership, error) {
	var out checkGroupMembership
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return out, fmt.Errorf("process table is unreadable: %w", err)
	}
	for _, entry := range entries {
		pid, convErr := strconv.Atoi(entry.Name())
		if convErr != nil || pid <= 0 || pid == leader {
			continue
		}
		data, readErr := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if readErr != nil {
			if errors.Is(readErr, fs.ErrNotExist) || errors.Is(readErr, syscall.ESRCH) {
				continue
			}
			return out, fmt.Errorf("process %d status is unreadable: %w", pid, readErr)
		}
		state, group, parseErr := parseProcStat(data)
		if parseErr != nil {
			return out, fmt.Errorf("process %d status is malformed", pid)
		}
		if group != pgid {
			continue
		}
		if state == 'Z' || state == 'X' {
			out.zombies++
			continue
		}
		out.live = append(out.live, pid)
	}
	return out, nil
}

// parseProcStat extracts the state and process-group id from /proc/<pid>/stat.
// The command name may contain spaces and parentheses, so fields are read after
// the last closing parenthesis.
func parseProcStat(data []byte) (byte, int, error) {
	text := string(data)
	end := strings.LastIndexByte(text, ')')
	if end < 0 {
		return 0, 0, errors.New("missing command name")
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 3 || len(fields[0]) != 1 {
		return 0, 0, errors.New("short status record")
	}
	group, err := strconv.Atoi(fields[2])
	if err != nil {
		return 0, 0, errors.New("invalid process group")
	}
	return fields[0][0], group, nil
}

// superviseCommand waits for the started command with the leader pinned. The
// trigger signals (timeout, output limit, interrupt escalation) reach the group
// while the leader is unreaped. The group gate settles before the reap, so no
// group signal is sent after reaping starts.
func (c *checkRun) superviseCommand(cmd *exec.Cmd, ctx context.Context, stdout, stderr *checkStream) commandOutcome {
	pid := cmd.Process.Pid
	exited := make(chan error, 1)
	go func() { exited <- observeLeaderExit(pid) }()

	var out commandOutcome
	var causes []string
	interrupts := c.interrupts
	ctxDone := ctx.Done()
	var grace <-chan time.Time
	killed := false
	kill := func() {
		if killed {
			return
		}
		killed = true
		if err := signalCheckTree(cmd, syscall.SIGKILL); err != nil {
			causes = append(causes, "cannot signal the owned process group: "+err.Error())
		}
	}
	var exitErr error
	pending := true
	for pending {
		select {
		case exitErr = <-exited:
			pending = false
		case <-ctxDone:
			ctxDone = nil
			kill()
		case observed := <-interrupts:
			interrupts = nil
			select {
			case exitErr = <-exited:
				// The command finished at the same moment as the signal.
				pending = false
			default:
				out.interrupted = true
				out.interruptSignal = observed
				if err := signalCheckTree(cmd, forwardedSignal(observed)); err != nil {
					causes = append(causes, "cannot forward the interrupt to the owned process group: "+err.Error())
				}
				grace = time.NewTimer(checkInterruptGrace).C
			}
		case <-grace:
			grace = nil
			kill()
		}
	}

	if exitErr != nil {
		// The leader's state cannot be observed, so no group signal is sent.
		out.group = uncertainGroup(exitErr)
		out.waitErr = cmd.Wait()
	} else {
		out.group = settlePinnedGroup(pid)
		out.waitErr = cmd.Wait()
		if !out.group.failed() {
			out.group = confirmGroupGone(pid)
		}
	}
	out.stdout = stdout.finish()
	out.stderr = stderr.finish()
	if len(causes) > 0 {
		out.group.cause = joinCauses(causes, out.group.cause)
	}
	return out
}
