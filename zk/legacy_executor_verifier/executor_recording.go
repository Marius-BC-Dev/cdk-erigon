package legacy_executor_verifier

import (
	"context"
	"fmt"
	"github.com/gateway-fm/cdk-erigon-lib/common"
	"github.com/ledgerwatch/erigon/zk/legacy_executor_verifier/proto/github.com/0xPolygonHermez/zkevm-node/state/runtime/executor"
	"github.com/ledgerwatch/log/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
	"os"
	"encoding/json"
	"path/filepath"
)

type ExecutorRecording struct {
	grpcUrl    string
	conn       *grpc.ClientConn
	connCancel context.CancelFunc
	client     executor.ExecutorServiceClient

	witnessFull bool
}

func NewExecutorsRecording(cfg Config, witnessFull bool) []*ExecutorRecording {
	executors := make([]*ExecutorRecording, len(cfg.GrpcUrls))
	var err error
	for i, grpcUrl := range cfg.GrpcUrls {
		executors[i], err = NewExecutorRecording(grpcUrl, cfg.Timeout, witnessFull)
		if err != nil {
			log.Warn("Failed to create executor", "error", err)
		}
	}
	return executors
}

func NewExecutorRecording(grpcUrl string, timeout time.Duration, witnessFull bool) (*ExecutorRecording, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	conn, err := grpc.DialContext(ctx, grpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to dial grpc: %w", err)
	}
	client := executor.NewExecutorServiceClient(conn)

	e := &ExecutorRecording{
		grpcUrl:     grpcUrl,
		conn:        conn,
		connCancel:  cancel,
		client:      client,
		witnessFull: witnessFull,
	}
	return e, nil
}

func (e *ExecutorRecording) Close() {
	if e == nil || e.conn == nil {
		return
	}
	e.connCancel()
	err := e.conn.Close()
	if err != nil {
		log.Warn("Failed to close grpc connection", err)
	}
}

func (e *ExecutorRecording) Verify(p *Payload, request *VerifierRequest) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.Debug("Sending request to grpc server", "grpcUrl", e.grpcUrl)

	size := 1024 * 1024 * 256 // 256mb maximum size - hack for now until trimmed witness is proved off
	resp, err := e.client.ProcessStatelessBatchV2(ctx, &executor.ProcessStatelessBatchRequestV2{
		Witness:           p.Witness,
		DataStream:        p.DataStream,
		Coinbase:          p.Coinbase,
		OldAccInputHash:   p.OldAccInputHash,
		L1InfoRoot:        p.L1InfoRoot,
		TimestampLimit:    p.TimestampLimit,
		ForcedBlockhashL1: p.ForcedBlockhashL1,
		ContextId:         p.ContextId,
		//TraceConfig: &executor.TraceConfigV2{
		//	DisableStorage:            0,
		//	DisableStack:              0,
		//	EnableMemory:              0,
		//	EnableReturnData:          0,
		//	TxHashToGenerateFullTrace: nil,
		//},
	}, grpc.MaxCallSendMsgSize(size), grpc.MaxCallRecvMsgSize(size))
	if err != nil {
		return false, fmt.Errorf("failed to process stateless batch: %w", err)
	}

	counters := fmt.Sprintf("[SHA: %v]", resp.CntSha256Hashes)
	counters += fmt.Sprintf("[A: %v]", resp.CntArithmetics)
	counters += fmt.Sprintf("[B: %v]", resp.CntBinaries)
	counters += fmt.Sprintf("[K: %v]", resp.CntKeccakHashes)
	counters += fmt.Sprintf("[M: %v]", resp.CntMemAligns)
	counters += fmt.Sprintf("[P: %v]", resp.CntPoseidonHashes)
	counters += fmt.Sprintf("[S: %v]", resp.CntSteps)
	counters += fmt.Sprintf("[D: %v]", resp.CntPoseidonPaddings)
	log.Info("executor result", "batch", request.BatchNumber, "counters", counters, "root", common.BytesToHash(resp.NewStateRoot), "our-root", request.StateRoot)
	log.Debug("Received response from executor", "grpcUrl", e.grpcUrl, "response", resp)

	// dump the request and response to the disk
	err = dumpRequestResponse(request, p, resp, e.witnessFull)
	return responseCheck(resp, request.StateRoot)
}

func dumpRequestResponse(request *VerifierRequest, payload *Payload, resp *executor.ProcessBatchResponseV2, witnessFull bool) error {
	dumpDir := "verifier_dumps"

	if err := os.MkdirAll(dumpDir, 0755); err != nil {
		return fmt.Errorf("failed to create dump directory: %w", err)
	}

	witnessPortion := "p"
	if witnessFull {
		witnessPortion = "f"
	}

	fileName := fmt.Sprintf("%d-batch_%d-witness_%s", time.Now().UnixMilli(), request.BatchNumber, witnessPortion)
	payloadFileName := filepath.Join(dumpDir, fileName+"-payload.json")
	requestFileName := filepath.Join(dumpDir, fileName+"-request.json")
	responseFileName := filepath.Join(dumpDir, fileName+"-response.json")

	pData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}

	rData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to serialize request: %w", err)
	}

	responseData, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}

	if err := os.WriteFile(payloadFileName, pData, 0644); err != nil {
		return fmt.Errorf("failed to write payload to file: %w", err)
	}

	if err := os.WriteFile(requestFileName, rData, 0644); err != nil {
		return fmt.Errorf("failed to write request to file: %w", err)
	}

	if err := os.WriteFile(responseFileName, responseData, 0644); err != nil {
		return fmt.Errorf("failed to write response to file: %w", err)
	}

	return nil
}
