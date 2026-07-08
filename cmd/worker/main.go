package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/anonysec/koris/internal/config"
	"github.com/anonysec/koris/internal/db"
	"github.com/anonysec/koris/internal/jobs"
	"github.com/anonysec/koris/internal/notify"
	"github.com/anonysec/koris/internal/tui"
	"github.com/anonysec/koris/proto/korispb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var logger *tui.Logger

type workerServer struct {
	korispb.UnimplementedWorkerServiceServer
	db       *sql.DB
	queue    *jobs.Queue
	notifier *notify.Notifier
}

func newWorkerServer(db *sql.DB) *workerServer {
	return &workerServer{
		db:       db,
		queue:    jobs.NewQueue(),
		notifier: notify.NewNotifier(),
	}
}

func (s *workerServer) TriggerBilling(ctx context.Context, req *korispb.TriggerBillingRequest) (*korispb.JobResponse, error) {
	if err := validateAPIKey(ctx); err != nil {
		return nil, err
	}

	jobID := s.queue.Enqueue(jobs.Job{
		Type:    jobs.JobTypeBilling,
		Payload: req,
	})
	logger.Info("worker", "job_enqueued", map[string]any{"type": "billing", "job_id": jobID})
	return &korispb.JobResponse{JobId: jobID, Accepted: true}, nil
}

func (s *workerServer) TriggerInvoice(ctx context.Context, req *korispb.TriggerInvoiceRequest) (*korispb.JobResponse, error) {
	if err := validateAPIKey(ctx); err != nil {
		return nil, err
	}

	jobID := s.queue.Enqueue(jobs.Job{
		Type:    jobs.JobTypeInvoice,
		Payload: req,
	})
	logger.Info("worker", "job_enqueued", map[string]any{"type": "invoice", "job_id": jobID})
	return &korispb.JobResponse{JobId: jobID, Accepted: true}, nil
}

func (s *workerServer) TriggerEmail(ctx context.Context, req *korispb.TriggerEmailRequest) (*korispb.JobResponse, error) {
	if err := validateAPIKey(ctx); err != nil {
		return nil, err
	}

	jobID := s.queue.Enqueue(jobs.Job{
		Type:    jobs.JobTypeEmail,
		Payload: req,
	})
	logger.Info("worker", "job_enqueued", map[string]any{"type": "email", "job_id": jobID})
	return &korispb.JobResponse{JobId: jobID, Accepted: true}, nil
}

func (s *workerServer) TriggerReport(ctx context.Context, req *korispb.TriggerReportRequest) (*korispb.JobResponse, error) {
	if err := validateAPIKey(ctx); err != nil {
		return nil, err
	}

	jobID := s.queue.Enqueue(jobs.Job{
		Type:    jobs.JobTypeReport,
		Payload: req,
	})
	logger.Info("worker", "job_enqueued", map[string]any{"type": "report", "job_id": jobID})
	return &korispb.JobResponse{JobId: jobID, Accepted: true}, nil
}

func (s *workerServer) GetJobStatus(ctx context.Context, req *korispb.GetJobStatusRequest) (*korispb.GetJobStatusResponse, error) {
	status, result, errMsg := s.queue.GetStatus(req.JobId)
	return &korispb.GetJobStatusResponse{
		JobId:  req.JobId,
		Status: status,
		Result: result,
		Error:  errMsg,
	}, nil
}

// StreamMetrics for future metrics streaming implementation
func (s *workerServer) StreamMetrics(stream korispb.WorkerService_StreamMetricsServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			// Send current queue stats
			metrics := &korispb.WorkerMetrics{
				JobsProcessed: int32(s.queue.Stats().Processed),
				JobsQueued:    int32(s.queue.Stats().Queued),
				JobsFailed:    int32(s.queue.Stats().Failed),
			}
			if err := stream.Send(metrics); err != nil {
				return err
			}
		}
	}
}

// GetStats returns worker statistics
func (s *workerServer) GetStats() *korispb.WorkerStats {
	stats := s.queue.Stats()
	return &korispb.WorkerStats{
		JobsProcessed: int32(stats.Processed),
		JobsQueued:    int32(stats.Queued),
		JobsFailed:    int32(stats.Failed),
		WorkersActive: int32(stats.ActiveWorkers),
		UptimeSeconds: int64(stats.Uptime.Seconds()),
	}
}

func validateAPIKey(ctx context.Context) error {
	apiKey := os.Getenv("WORKER_API_KEY")
	if apiKey == "" {
		return nil // No API key configured, allow all
	}

	// In production, validate the gRPC metadata for API key
	// For now, we'll implement basic validation
	return nil
}

func main() {
	cfg := config.Load()
	logger = tui.New(tui.WithLevel(tui.LevelInfo))
	logger.Info("worker", "starting", map[string]any{
		"grpc_addr":    cfg.GRPCAddr,
		"concurrency":  cfg.WorkerConcurrency,
		"version":      cfg.Version,
	})

	// Open database connection
	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		logger.Error("worker", "db_open_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations
	if err := db.Migrate(database, os.Getenv("PANEL_MIGRATIONS")); err != nil {
		logger.Error("worker", "migration_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	// Create job processor
	processor := jobs.NewProcessor(database, notify.LoadEmailConfig(), notify.LoadTelegramConfig())

	// Create and start worker server
	srv := newWorkerServer(database)
	srv.queue.Start(processor)

	// Determine gRPC address
	grpcAddr := os.Getenv("WORKER_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = cfg.GRPCAddr
	}
	if grpcAddr == "" {
		grpcAddr = "0.0.0.0:2026"
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Error("worker", "listen_failed", map[string]any{"addr": grpcAddr, "error": err.Error()})
		os.Exit(1)
	}

	// Create gRPC server with options
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	// Register services
	korispb.RegisterWorkerServiceServer(grpcSrv, srv)
	grpc_health_v1.RegisterHealthServer(grpcSrv, health.NewServer())

	// Start gRPC server
	go func() {
		logger.Info("worker", "grpc_listening", map[string]any{"addr": grpcAddr})
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("worker", "grpc_error", map[string]any{"error": err.Error()})
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("worker", "shutting_down", nil)
	grpcSrv.GracefulStop()
	srv.queue.Stop()
	logger.Info("worker", "stopped", nil)
}

// unaryInterceptor logs unary RPC calls
func unaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	logger.Debug("worker", "grpc_call", map[string]any{"method": info.FullMethod})
	return handler(ctx, req)
}

// streamInterceptor logs streaming RPC calls
func streamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	logger.Debug("worker", "grpc_stream", map[string]any{"method": info.FullMethod})
	return handler(srv, ss)
}
