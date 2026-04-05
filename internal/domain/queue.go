package domain

import "time"

type QueueEntryStatus string

const (
	QueueEntryStatusWaiting   QueueEntryStatus = "WAITING"
	QueueEntryStatusAdmitted  QueueEntryStatus = "ADMITTED"
	QueueEntryStatusCompleted QueueEntryStatus = "COMPLETED"
	QueueEntryStatusExpired   QueueEntryStatus = "EXPIRED"
	QueueEntryStatusAbandoned QueueEntryStatus = "ABANDONED"
)

// QueueEntryTTL is how long a user can wait in queue before
// their entry expires. After this, they must rejoin.
const QueueEntryTTL = 2 * time.Hour

// AdmissionWindowTTL is how long an admitted user has to claim
// before their admission token expires.
const AdmissionWindowTTL = 5 * time.Minute

type QueueEntry struct {
	ID         string
	EventID    string
	UserID     string
	Position   int64 // ZRANK from Redis, populated at load time
	Status     QueueEntryStatus
	JoinedAt   time.Time
	AdmittedAt *time.Time // nil until admitted
	UpdatedAt  time.Time
}

// IsExpired returns true if a WAITING entry has exceeded the
// queue TTL, or an ADMITTED entry has exceeded the admission window.
func (q *QueueEntry) IsExpired() bool {
	switch q.Status { //nolint:exhaustive // only check against concerned states
	case QueueEntryStatusWaiting:
		return time.Since(q.JoinedAt) > QueueEntryTTL
	case QueueEntryStatusAdmitted:
		if q.AdmittedAt == nil {
			return false
		}
		return time.Since(*q.AdmittedAt) > AdmissionWindowTTL
	default:
		return false
	}
}

// Admit transitions the entry from WAITING to ADMITTED.
func (q *QueueEntry) Admit() error {
	if q.Status != QueueEntryStatusWaiting {
		return ErrInvalidTransition
	}
	if q.IsExpired() {
		return ErrQueueEntryExpired
	}
	now := time.Now()
	q.AdmittedAt = &now
	q.Status = QueueEntryStatusAdmitted
	q.UpdatedAt = now
	return nil
}

// Complete transitions the entry from ADMITTED to COMPLETED.
// Called when the user successfully claims a ticket.
func (q *QueueEntry) Complete() error {
	if q.Status != QueueEntryStatusAdmitted {
		return ErrInvalidTransition
	}
	q.Status = QueueEntryStatusCompleted
	q.UpdatedAt = time.Now()
	return nil
}

// Abandon transitions the entry from WAITING to ABANDONED.
// Called when the user explicitly leaves the queue.
func (q *QueueEntry) Abandon() error {
	if q.Status != QueueEntryStatusWaiting {
		return ErrInvalidTransition
	}
	q.Status = QueueEntryStatusAbandoned
	q.UpdatedAt = time.Now()
	return nil
}

// Expire transitions the entry to EXPIRED from either
// WAITING or ADMITTED state.
func (q *QueueEntry) Expire() error {
	if q.Status != QueueEntryStatusWaiting && q.Status != QueueEntryStatusAdmitted {
		return ErrInvalidTransition
	}
	q.Status = QueueEntryStatusExpired
	q.UpdatedAt = time.Now()
	return nil
}
