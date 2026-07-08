package korispb

import (
	context "context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Message types for WorkerService

type TriggerBillingRequest struct {
	Username    string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	PeriodStart string `protobuf:"bytes,2,opt,name=period_start,json=periodStart,proto3" json:"periodStart,omitempty"`
	PeriodEnd   string `protobuf:"bytes,3,opt,name=period_end,json=periodEnd,proto3" json:"periodEnd,omitempty"`
}

type TriggerInvoiceRequest struct {
	CustomerId  int64  `protobuf:"varint,1,opt,name=customer_id,json=customerId,proto3" json:"customerId,omitempty"`
	PeriodStart string `protobuf:"bytes,2,opt,name=period_start,json=periodStart,proto3" json:"periodStart,omitempty"`
	PeriodEnd   string `protobuf:"bytes,3,opt,name=period_end,json=periodEnd,proto3" json:"periodEnd,omitempty"`
}

type TriggerEmailRequest struct {
	To       string `protobuf:"bytes,1,opt,name=to,proto3" json:"to,omitempty"`
	Subject  string `protobuf:"bytes,2,opt,name=subject,proto3" json:"subject,omitempty"`
	Body     string `protobuf:"bytes,3,opt,name=body,proto3" json:"body,omitempty"`
	Template string `protobuf:"bytes,4,opt,name=template,proto3" json:"template,omitempty"`
}

type TriggerReportRequest struct {
	ReportType  string `protobuf:"bytes,1,opt,name=report_type,json=reportType,proto3" json:"reportType,omitempty"`
	PeriodStart string `protobuf:"bytes,2,opt,name=period_start,json=periodStart,proto3" json:"periodStart,omitempty"`
	PeriodEnd   string `protobuf:"bytes,3,opt,name=period_end,json=periodEnd,proto3" json:"periodEnd,omitempty"`
	AdminId     int64  `protobuf:"varint,4,opt,name=admin_id,json=adminId,proto3" json:"adminId,omitempty"`
}

type GetJobStatusRequest struct {
	JobId string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"jobId,omitempty"`
}

type GetJobStatusResponse struct {
	JobId    string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"jobId,omitempty"`
	Status   string `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
	Result   string `protobuf:"bytes,3,opt,name=result,proto3" json:"result,omitempty"`
	Error    string `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
	Progress int64  `protobuf:"varint,5,opt,name=progress,proto3" json:"progress,omitempty"`
}

type JobResponse struct {
	JobId    string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"jobId,omitempty"`
	Accepted bool   `protobuf:"varint,2,opt,name=accepted,proto3"  json:"accepted,omitempty"`
}

type WorkerMetrics struct {
	JobsProcessed int32   `protobuf:"varint,1,opt,name=jobs_processed,json=jobsProcessed,proto3" json:"jobsProcessed,omitempty"`
	JobsQueued    int32   `protobuf:"varint,2,opt,name=jobs_queued,json=jobsQueued,proto3" json:"jobsQueued,omitempty"`
	JobsFailed    int32   `protobuf:"varint,3,opt,name=jobs_failed,json=jobsFailed,proto3" json:"jobsFailed,omitempty"`
	CpuUsage      float64 `protobuf:"fixed64,4,opt,name=cpu_usage,json=cpuUsage,proto3" json:"cpuUsage,omitempty"`
	MemUsageBytes int64   `protobuf:"varint,5,opt,name=mem_usage_bytes,json=memUsageBytes,proto3" json:"memUsageBytes,omitempty"`
}

type WorkerAck struct {
	Ok bool `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
}

// WorkerService client interface (implemented by generated gRPC code)
type WorkerServiceClient interface {
	TriggerBilling(ctx context.Context, in *TriggerBillingRequest, opts ...grpc.CallOption) (*JobResponse, error)
	TriggerInvoice(ctx context.Context, in *TriggerInvoiceRequest, opts ...grpc.CallOption) (*JobResponse, error)
	TriggerEmail(ctx context.Context, in *TriggerEmailRequest, opts ...grpc.CallOption) (*JobResponse, error)
	TriggerReport(ctx context.Context, in *TriggerReportRequest, opts ...grpc.CallOption) (*JobResponse, error)
	GetJobStatus(ctx context.Context, in *GetJobStatusRequest, opts ...grpc.CallOption) (*GetJobStatusResponse, error)
}

// WorkerService server interface
type WorkerServiceServer interface {
	TriggerBilling(context.Context, *TriggerBillingRequest) (*JobResponse, error)
	TriggerInvoice(context.Context, *TriggerInvoiceRequest) (*JobResponse, error)
	TriggerEmail(context.Context, *TriggerEmailRequest) (*JobResponse, error)
	TriggerReport(context.Context, *TriggerReportRequest) (*JobResponse, error)
	GetJobStatus(context.Context, *GetJobStatusRequest) (*GetJobStatusResponse, error)
}