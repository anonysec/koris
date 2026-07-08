package main

import (
	"context"

	"github.com/anonysec/koris/internal/jobs"
	"github.com/anonysec/koris/proto/korispb"
)

// Stats returns queue statistics
func (s *workerServer) Stats() *jobs.Stats {
	return s.queue.Stats()
}

// GetWorkerStats implements the RPC for getting worker statistics
func (s *workerServer) GetWorkerStats(ctx context.Context, req *korispb.GetWorkerStatsRequest) (*korispb.WorkerStatsResponse, error) {
	stats := s.Stats()
	return &korispb.WorkerStatsResponse{
		JobsProcessed: int32(stats.Processed),
		JobsQueued:    int32(stats.Queued),
		JobsFailed:    int32(stats.Failed),
		WorkersActive: int32(stats.ActiveWorkers),
		UptimeSeconds: int64(stats.Uptime.Seconds()),
	}, nil
}

// CancelJob allows cancellation of queued jobs
func (s *workerServer) CancelJob(ctx context.Context, req *korispb.CancelJobRequest) (*korispb.CancelJobResponse, error) {
	success := s.queue.Cancel(req.JobId)
	return &korispb.CancelJobResponse{
		JobId:    req.JobId,
		Cancelled: success,
	}, nil
}
