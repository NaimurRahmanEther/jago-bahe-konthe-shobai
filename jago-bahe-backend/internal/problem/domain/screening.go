package domain

// RejectionReason is the ground an admin rejected a problem on. It is a closed
// enum by design (Concept §4): an admin may reject a report for not being a
// genuine report, but never on merit — whether a genuine problem is real, local,
// and worth an official's time stays the community's call via the threshold V.
// Widening this enum widens the takedown veto, so treat additions as a change to
// the platform's power balance, not a convenience.
type RejectionReason string

const (
	ReasonSpam      RejectionReason = "spam"
	ReasonAbusive   RejectionReason = "abusive"
	ReasonDuplicate RejectionReason = "duplicate"
	ReasonWrongArea RejectionReason = "wrong_area"
)

// Valid reports whether r is one of the four permitted grounds.
func (r RejectionReason) Valid() bool {
	switch r {
	case ReasonSpam, ReasonAbusive, ReasonDuplicate, ReasonWrongArea:
		return true
	}
	return false
}

// Screening is the authority for the pre-publication gate: whether a report may be
// approved into public view, and whether a takedown is still legal and on what
// ground. The http/UI layers only display the outcome.
//
// Every new problem starts PendingApproval — invisible to all but its reporter and
// its union's admin — and is published (StatusReported) only when that admin
// approves it. There is deliberately no auto-publish: an admin who never acts
// leaves the report hidden. An earlier design published on report and moderated
// only afterwards; it was reversed because spam and defamation were publicly
// readable before any human saw them. The cost accepted in return is admin
// inaction, which this design treats as the admin's accountable choice, not the
// platform's silent default. The takedown that remains is narrow on purpose (see
// Status.Rejectable) and always public (see StatusRejected).
type Screening struct{}

// NewScreening constructs the screening domain service.
func NewScreening() *Screening { return &Screening{} }

// Approve publishes a pending report: the union's admin has screened it and found
// it to be a genuine report. It becomes Reported — public, but still unvalidated,
// since clearing the spam gate says nothing about merit. From here the community
// decides via V. Approving anything that is not PendingApproval is ErrNotPending.
func (Screening) Approve(current Status) (Status, error) {
	if current != StatusPendingApproval {
		return "", ErrNotPending
	}
	return StatusReported, nil
}

// Reject takes a problem down on one of the fixed grounds — either during
// screening (from PendingApproval) or as a post-publication takedown (from
// Reported/Validated, before assignment). Status is checked before the reason so
// that an admin acting on a problem past the rejectable window is told the real
// obstacle rather than being sent to fix their reason.
func (Screening) Reject(current Status, reason RejectionReason) (Status, error) {
	if !current.Rejectable() {
		return "", ErrNotRejectable
	}
	if !reason.Valid() {
		return "", ErrInvalidRejectionReason
	}
	return StatusRejected, nil
}
