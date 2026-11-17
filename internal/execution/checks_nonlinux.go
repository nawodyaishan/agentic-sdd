//go:build !linux

package execution

// Classification-only completion for hosts without a pinned leader observation
// (darwin, other Unix, Windows). The leader is reaped by cmd.Wait as soon as it
// exits, so the owned group cannot be enumerated. Trigger signals (timeout,
// output limit, interrupt escalation) may still be sent while the leader has not
// been reaped. After the reap, only kill(-pgid, 0) probes run: no signal is sent
// and a surviving group fails the check. A narrow window remains in which the
// leader is reaped while a trigger is being sent; that residual is documented
// rather than claimed as eliminated.

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// listCheckGroupMembers cannot enumerate the owned group on this platform.
func listCheckGroupMembers(pgid, leader int) (checkGroupMembership, error) {
	return checkGroupMembership{}, errGroupMembersUnobservable
}

// superviseCommand waits for the started command and classifies the owned group
// after the reap.
func (c *checkRun) superviseCommand(cmd *exec.Cmd, ctx context.Context, stdout, stderr *checkStream) commandOutcome {
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	var out commandOutcome
	var causes []string
	var waitErr error
	reaped := false
	reap := func(err error) {
		waitErr = err
		reaped = true
	}
	// nonBlockingReap consumes the reap result when it is already available, so a
	// trigger is never sent once the leader has been reaped.
	nonBlockingReap := func() bool {
		select {
		case err := <-waitCh:
			reap(err)
			return true
		default:
			return false
		}
	}
	interrupts := c.interrupts
	ctxDone := ctx.Done()
	var grace <-chan time.Time
	killed := false
	kill := func() {
		if killed || nonBlockingReap() {
			return
		}
		killed = true
		if err := signalCheckTree(cmd, syscall.SIGKILL); err != nil {
			causes = append(causes, "cannot signal the owned process group: "+err.Error())
		}
	}
	for !reaped {
		select {
		case err := <-waitCh:
			reap(err)
		case <-ctxDone:
			ctxDone = nil
			kill()
		case observed := <-interrupts:
			interrupts = nil
			if nonBlockingReap() {
				break
			}
			out.interrupted = true
			out.interruptSignal = observed
			if err := signalCheckTree(cmd, forwardedSignal(observed)); err != nil {
				causes = append(causes, "cannot forward the interrupt to the owned process group: "+err.Error())
			}
			grace = time.NewTimer(checkInterruptGrace).C
		case <-grace:
			grace = nil
			kill()
		}
	}

	// The leader is reaped. Classification only: probes, no signal.
	out.waitErr = waitErr
	out.group = confirmGroupGone(cmd.Process.Pid)
	out.stdout = stdout.finish()
	out.stderr = stderr.finish()
	if len(causes) > 0 {
		out.group.cause = joinCauses(causes, out.group.cause)
	}
	return out
}
