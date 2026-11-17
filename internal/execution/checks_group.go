package execution

// Owned process-group completion (Feature 022 B2, FR-01/O-06, plan §5 and §12.2).
//
// A check is owned by its process group: the leader is started with
// Setpgid, so the group id equals the leader pid. Group liveness is decided from
// that group id while the leader is still unreaped, because an unreaped leader
// pins its pid and therefore the group id. Reaping starts only after the group
// gate has closed. A passing check requires the leader exit 0, no surviving
// owned member after the bounded grace, and kill(-pgid, 0) reporting ESRCH after
// the reap. Any uncertainty fails the check and sends no further signal.

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// checkGroupMembership is one observation of the owned group. Live members are
// listed by pid; zombies are exited members not yet reaped by their parent.
type checkGroupMembership struct {
	live    []int
	zombies int
}

// Post-exit bounds. These are the production values; tests shorten them through
// the variables, and no fixture waits out a production window.
var (
	checkPostExitGrace = 2 * time.Second
	checkTermWait      = 1 * time.Second
	checkGroupPoll     = 10 * time.Millisecond
)

// Group hooks. Production wires them to the platform primitives; tests replace
// them to inject uncertainty and to record every group signal.
var (
	checkGroupMembers = listCheckGroupMembers
	checkGroupSignal  = sendCheckGroupSignal
	checkGroupProbe   = probeCheckGroup
)

// errGroupMembersUnobservable reports that the platform cannot enumerate the
// owned group, so completion cannot be proven.
var errGroupMembersUnobservable = errors.New("process group membership is not observable on this platform")

// commandOutcome is the observed end of one started command.
type commandOutcome struct {
	waitErr         error
	stdout          checkCapture
	stderr          checkCapture
	interrupted     bool
	interruptSignal os.Signal
	group           groupVerdict
}

// groupVerdict is the group gate result. An empty cause means the gate passed.
type groupVerdict struct {
	cause     string
	survivors bool
	uncertain bool
}

// failed reports whether the gate refuses a passing result.
func (v groupVerdict) failed() bool {
	return v.cause != ""
}

// uncertainGroup refuses completion without sending any further signal.
func uncertainGroup(err error) groupVerdict {
	detail := "unknown error"
	if err != nil {
		detail = sanitizeCheckText([]byte(err.Error()), checkSummaryLineBytes)
	}
	return groupVerdict{
		cause:     "owned process group could not be observed (" + detail + "); no further signal was sent",
		uncertain: true,
	}
}

// waitGroupEmpty polls for live owned members until none remain or limit passes.
// Zombies do not count: they hold no running work.
func waitGroupEmpty(pgid int, limit time.Duration) (bool, error) {
	deadline := time.Now().Add(limit)
	for {
		members, err := checkGroupMembers(pgid, pgid)
		if err != nil {
			return false, err
		}
		if len(members.live) == 0 {
			return true, nil
		}
		if !time.Now().Before(deadline) {
			return false, nil
		}
		time.Sleep(checkGroupPoll)
	}
}

// settlePinnedGroup runs while the leader is unreaped. Survivors get the
// post-exit grace to finish on their own; otherwise they are terminated with
// SIGTERM, then SIGKILL, all before the reap. Survivors always fail the gate.
func settlePinnedGroup(pgid int) groupVerdict {
	empty, err := waitGroupEmpty(pgid, checkPostExitGrace)
	if err != nil {
		return uncertainGroup(err)
	}
	if empty {
		return groupVerdict{}
	}
	if err := checkGroupSignal(pgid, syscall.SIGTERM); err != nil {
		return uncertainGroup(err)
	}
	empty, err = waitGroupEmpty(pgid, checkTermWait)
	if err != nil {
		return uncertainGroup(err)
	}
	if !empty {
		if err := checkGroupSignal(pgid, syscall.SIGKILL); err != nil {
			return uncertainGroup(err)
		}
		empty, err = waitGroupEmpty(pgid, checkTermWait)
		if err != nil {
			return uncertainGroup(err)
		}
		if !empty {
			return groupVerdict{cause: "owned process group still had active children after the command exited and survived SIGKILL", survivors: true}
		}
	}
	return groupVerdict{cause: "owned process group still had active children after the command exited; they were terminated by escalation", survivors: true}
}

// confirmGroupGone runs after the reap. The group id is passing only when the
// probe reports it gone within the grace. A probe failure is uncertainty.
func confirmGroupGone(pgid int) groupVerdict {
	deadline := time.Now().Add(checkPostExitGrace)
	for {
		alive, err := checkGroupProbe(pgid)
		if err != nil {
			return uncertainGroup(err)
		}
		if !alive {
			return groupVerdict{}
		}
		if !time.Now().Before(deadline) {
			return groupVerdict{cause: "owned process group still existed after the command was reaped", survivors: true}
		}
		time.Sleep(checkGroupPoll)
	}
}

// forwardedSignal returns the signal to pass to the owned group. A signal that
// is not a Unix signal is forwarded as SIGTERM.
func forwardedSignal(sig os.Signal) os.Signal {
	if _, ok := sig.(syscall.Signal); ok {
		return sig
	}
	return syscall.SIGTERM
}

// signalLeaderFallback kills only the unreaped leader when the group signal
// cannot be sent. The leader pid is certain while it is unreaped.
func signalLeaderFallback(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// joinCauses joins observed causes into one bounded text.
func joinCauses(causes []string, tail string) string {
	if tail != "" {
		causes = append(causes, tail)
	}
	return strings.Join(causes, "; ")
}
