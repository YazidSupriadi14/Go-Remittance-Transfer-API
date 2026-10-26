package transfer

// Status is the lifecycle state of a transfer.
//
//	REQUESTED ──► SENDING ──► COMPLETED
//	    │            └──────► FAILED
//	    └──► CANCELLED
type Status string

const (
	StatusRequested Status = "REQUESTED"
	StatusSending   Status = "SENDING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

// allowedTransitions is the single source of truth for the state machine.
// Adding a new transition means adding one entry here; no other code changes.
var allowedTransitions = map[Status][]Status{
	StatusRequested: {StatusSending, StatusCancelled},
	StatusSending:   {StatusCompleted, StatusFailed},
}

// CanTransitionTo reports whether moving from s to next is allowed.
func (s Status) CanTransitionTo(next Status) bool {
	for _, allowed := range allowedTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}
