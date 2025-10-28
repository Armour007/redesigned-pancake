package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	// drainMode prevents readers from consuming new messages; reclaimer continues to finish pending
	drainMode bool
	// drainedComplete becomes true after drainMode is enabled and pending == 0 for N consecutive ticks
	drainedComplete       bool
	drainZeroPendingTicks int
)

const codegenStream = "aura:jobs:codegen"
const codegenGroup = "codegen"
const codegenDLQStream = "aura:jobs:codegen:dlq"

func initRedisFromEnv() bool {
	if os.Getenv("AURA_QUEUE_ENABLE") == "" {
		return false
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	return true
}

// StartCodegenWorker starts a simple Redis Streams consumer in-background.
func StartCodegenWorker(ctx context.Context) {
	if !initRedisFromEnv() {
		return
	}
	// Log worker online and pending summary for runbooks
	if p, err := redisClient.XPending(ctx, codegenStream, codegenGroup).Result(); err == nil && p != nil {
		log.Printf("queue worker online: pending=%d", p.Count)
	} else {
		log.Printf("queue worker online: pending=unknown (group may be new)")
	}
	// Ensure consumer group exists
	_ = redisClient.XGroupCreateMkStream(ctx, codegenStream, codegenGroup, "$").Err()
	// Worker pool size
	workers := 2
	if v := os.Getenv("AURA_QUEUE_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}
	// Read batch size
	readCount := 4
	if v := os.Getenv("AURA_QUEUE_READ_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			readCount = n
		}
	}
	// Optional global rate limit (per second)
	var rateTicker *time.Ticker
	if v := os.Getenv("AURA_QUEUE_RATE_PER_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateTicker = time.NewTicker(time.Second / time.Duration(n))
		}
	}

	// Start workers (distinct consumer names)
	for i := 0; i < workers; i++ {
		consumer := fmt.Sprintf("worker-%d-%d", time.Now().UnixNano(), i)
		go func(consumerName string) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if drainMode {
					// In drain, skip reading new items; let reclaimer finish pending
					time.Sleep(500 * time.Millisecond)
					continue
				}
				streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    codegenGroup,
					Consumer: consumerName,
					Streams:  []string{codegenStream, ">"},
					Count:    int64(readCount),
					Block:    5 * time.Second,
				}).Result()
				if err != nil && err != redis.Nil {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				for _, s := range streams {
					for _, msg := range s.Messages {
						if rateTicker != nil {
							select {
							case <-ctx.Done():
								return
							case <-rateTicker.C:
							}
						}
						ack := processCodegenMessage(ctx, msg)
						if ack {
							_, _ = redisClient.XAck(ctx, codegenStream, codegenGroup, msg.ID).Result()
						}
					}
				}
			}
		}(consumer)
	}

	// Reclaimer: scans pending and reclaims or DLQs
	minIdle := 30 * time.Second
	if v := os.Getenv("AURA_QUEUE_PENDING_IDLE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			minIdle = time.Duration(ms) * time.Millisecond
		}
	}
	maxDeliveries := 3
	if v := os.Getenv("AURA_QUEUE_MAX_DELIVERIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxDeliveries = n
		}
	}
	scanEvery := 10 * time.Second
	if v := os.Getenv("AURA_QUEUE_RECLAIM_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			scanEvery = time.Duration(ms) * time.Millisecond
		}
	}
	// Drain empty threshold (consecutive zero-pending ticks required to mark drained)
	emptyThresh := 3
	if v := os.Getenv("AURA_QUEUE_DRAIN_EMPTY_TICKS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			emptyThresh = n
		}
	}
	batch := 10
	if v := os.Getenv("AURA_QUEUE_AUTOCLAIM_BATCH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			batch = n
		}
	}
	reclaimer := fmt.Sprintf("reclaimer-%d", time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(scanEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			// Update a summary pending gauge for alerting and drain progression
			if p, err := redisClient.XPending(ctx, codegenStream, codegenGroup).Result(); err == nil && p != nil {
				SetQueuePending("codegen", p.Count)
				if drainMode {
					if p.Count == 0 {
						drainZeroPendingTicks++
						if drainZeroPendingTicks >= emptyThresh {
							drainedComplete = true
						}
					} else {
						drainZeroPendingTicks = 0
						drainedComplete = false
					}
				} else {
					// not draining; reset
					drainZeroPendingTicks = 0
					drainedComplete = false
				}
			}
			// Inspect pending with details to get delivery counts
			pendings, err := redisClient.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream: codegenStream,
				Group:  codegenGroup,
				Start:  "-",
				End:    "+",
				Count:  int64(batch),
			}).Result()
			if err != nil || len(pendings) == 0 {
				continue
			}
			for _, p := range pendings {
				if p.Idle < minIdle {
					continue
				}
				if int(p.RetryCount) >= maxDeliveries {
					// Move to DLQ
					// Fetch message to capture payload
					msgs, _ := redisClient.XRange(ctx, codegenStream, p.ID, p.ID).Result()
					var payload any = map[string]any{"error": "missing"}
					if len(msgs) == 1 {
						payload = msgs[0].Values["payload"]
					}
					_, _ = redisClient.XAdd(ctx, &redis.XAddArgs{
						Stream: codegenDLQStream,
						Values: map[string]any{
							"payload":    payload,
							"reason":     fmt.Sprintf("max deliveries %d exceeded", maxDeliveries),
							"deliveries": p.RetryCount,
							"at":         time.Now().Unix(),
						},
					}).Result()
					RecordDLQInsert("codegen", "max_deliveries_exceeded")
					if xlen, err := redisClient.XLen(ctx, codegenDLQStream).Result(); err == nil {
						SetDLQDepth("codegen", xlen)
					}
					// Ack original to drop it
					_, _ = redisClient.XAck(ctx, codegenStream, codegenGroup, p.ID).Result()
					continue
				}
				// Claim and process
				claimed, err := redisClient.XClaim(ctx, &redis.XClaimArgs{
					Stream:   codegenStream,
					Group:    codegenGroup,
					Consumer: reclaimer,
					MinIdle:  minIdle,
					Messages: []string{p.ID},
				}).Result()
				if err != nil || len(claimed) == 0 {
					continue
				}
				for _, msg := range claimed {
					ack := processCodegenMessage(ctx, msg)
					if ack {
						_, _ = redisClient.XAck(ctx, codegenStream, codegenGroup, msg.ID).Result()
					}
				}
			}
		}
	}()
}

// EnqueueCodegen publishes a codegen job to Redis Streams
func EnqueueCodegen(job *sdkJob) error {
	if redisClient == nil {
		return fmt.Errorf("queue disabled")
	}
	b, _ := json.Marshal(job)
	return redisClient.XAdd(context.Background(), &redis.XAddArgs{
		Stream: codegenStream,
		Values: map[string]any{"payload": string(b)},
	}).Err()
}

// processCodegenMessage handles idempotence, executes codegen, and decides whether to ack.
// Returns true when the message should be ACKed; false when it should be left pending for retry/reclaim.
func processCodegenMessage(ctx context.Context, msg redis.XMessage) bool {
	// Parse job
	var job sdkJob
	if payload, ok := msg.Values["payload"].(string); ok {
		_ = json.Unmarshal([]byte(payload), &job)
	} else {
		// malformed; drop
		return true
	}

	// Seed job map if missing
	sdkJobs.mu.Lock()
	if _, ok := sdkJobs.jobs[job.ID]; !ok {
		sdkJobs.jobs[job.ID] = &sdkJob{ID: job.ID, Lang: job.Lang, AgentID: job.AgentID, OrganizationID: job.OrganizationID, Email: job.Email, Status: "queued"}
	}
	// Idempotence: if already ready or file exists, skip
	existing := sdkJobs.jobs[job.ID]
	file := sdkJobs.files[job.ID]
	_, inMem := sdkJobs.zips[job.ID]
	sdkJobs.mu.Unlock()

	if existing != nil && existing.Status == "ready" {
		return true
	}
	if file != "" {
		// Verify on disk exists (defensive)
		if _, err := os.Stat(filepath.Clean(file)); err == nil {
			return true
		}
	}
	if inMem {
		return true
	}

	// Execute codegen
	runCodegen(&job)

	// Ack only on success; leave pending to be reclaimed on error
	sdkJobs.mu.Lock()
	defer sdkJobs.mu.Unlock()
	if j, ok := sdkJobs.jobs[job.ID]; ok {
		return j.Status == "ready"
	}
	// If no record, safest is to retry
	return false
}
