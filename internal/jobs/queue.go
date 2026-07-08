package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/anonysec/koris/internal/notify"
	"github.com/anonysec/koris/proto/korispb"
)

const (
	JobTypeBilling = "billing"
	JobTypeInvoice = "invoice"
	JobTypeEmail   = "email"
	JobTypeReport  = "report"
)

type Job struct {
	ID        string
	Type      string
	Payload   any
	CreatedAt time.Time
	Status    string
	Result    string
	Error     string
	StartedAt *time.Time
	Cancelled bool
}

type processor interface {
	Process(Job) (string, error)
}

type jobStatus struct {
	status string
	result string
	errMsg string
}

type Queue struct {
	jobs           map[string]*Job
	statuses       map[string]*jobStatus
	mu             sync.RWMutex
	workCh         chan *Job
	done           chan struct{}
	processor
	startedAt     time.Time
	processed     int64
	failed        int64
	activeWorkers int32
}

func NewQueue() *Queue {
	return &Queue{
		jobs:     make(map[string]*Job),
		statuses: make(map[string]*jobStatus),
		workCh:   make(chan *Job, 100),
		done:     make(chan struct{}),
	}
}

func (q *Queue) Start(p processor) {
	q.processor = p
	q.startedAt = time.Now()
	for i := 0; i < 4; i++ {
		go q.worker(i)
	}
}

func (q *Queue) worker(id int) {
	q.ActiveWorker()
	defer q.InactiveWorker()

	for {
		select {
		case <-q.done:
			return
		case job := <-q.workCh:
			q.mu.Lock()
			job.Status = "running"
			now := time.Now()
			job.StartedAt = &now
			q.mu.Unlock()

			result, err := q.process(*job)
			status := "completed"
			errMsg := ""
			if err != nil {
				status = "failed"
				errMsg = err.Error()
				q.Failed()
			} else {
				q.Processed()
			}

			q.mu.Lock()
			job.Status = status
			job.Result = result
			job.Error = errMsg
			q.statuses[job.ID] = &jobStatus{status: status, result: result, errMsg: errMsg}
			q.mu.Unlock()
		}
	}
}

func (q *Queue) Enqueue(job Job) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d-%s", time.Now().UnixNano(), job.Type)
	}
	job.CreatedAt = time.Now()
	job.Status = "queued"

	q.jobs[job.ID] = &job
	q.statuses[job.ID] = &jobStatus{status: "queued"}

	go func() {
		q.workCh <- &job
	}()

	return job.ID
}

func (q *Queue) GetStatus(id string) (status, result, errMsg string) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if s, ok := q.statuses[id]; ok {
		return s.status, s.result, s.errMsg
	}
	return "", "", "job not found"
}

func (q *Queue) Cancel(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if job, ok := q.jobs[id]; ok && job.Status == "queued" && !job.Cancelled {
		job.Cancelled = true
		job.Status = "cancelled"
		q.statuses[id] = &jobStatus{status: "cancelled"}
		return true
	}
	return false
}

func (q *Queue) Stop() {
	close(q.done)
}

func (q *Queue) process(job Job) (string, error) {
	if p, ok := q.processor.(interface{ ProcessWithContext(context.Context, Job) (string, error) }); ok {
		return p.ProcessWithContext(context.Background(), job)
	}
	return q.processor.Process(job)
}

// ProcessorWithContext is a processor that supports context
type ProcessorWithContext interface {
	ProcessWithContext(ctx context.Context, job Job) (string, error)
}

// ProcessWithContext processes a job with context support
func (p *Processor) ProcessWithContext(ctx context.Context, job Job) (string, error) {
	switch job.Type {
	case JobTypeBilling:
		req, ok := job.Payload.(*korispb.TriggerBillingRequest)
		if !ok {
			return "", fmt.Errorf("invalid billing payload")
		}
		return p.processBilling(ctx, req)
	case JobTypeInvoice:
		req, ok := job.Payload.(*korispb.TriggerInvoiceRequest)
		if !ok {
			return "", fmt.Errorf("invalid invoice payload")
		}
		return p.processInvoice(ctx, req)
	case JobTypeEmail:
		req, ok := job.Payload.(*korispb.TriggerEmailRequest)
		if !ok {
			return "", fmt.Errorf("invalid email payload")
		}
		return p.processEmail(ctx, req)
	case JobTypeReport:
		req, ok := job.Payload.(*korispb.TriggerReportRequest)
		if !ok {
			return "", fmt.Errorf("invalid report payload")
		}
		return p.processReport(ctx, req)
	default:
		return "", fmt.Errorf("unknown job type: %s", job.Type)
	}
}
