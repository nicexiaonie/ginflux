package ginflux

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Writer 批量写入器
type Writer struct {
	client    *Client
	bucket    string
	batchSize int
	interval  time.Duration
	buffer    []*write.Point
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	errChan   chan error
	closed    bool
}

// NewWriter 创建批量写入器
func NewWriter(client *Client, bucket string, batchSize int, interval time.Duration) *Writer {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Writer{
		client:    client,
		bucket:    bucket,
		batchSize: batchSize,
		interval:  interval,
		buffer:    make([]*write.Point, 0, batchSize),
		ctx:       ctx,
		cancel:    cancel,
		errChan:   make(chan error, 100),
	}

	w.wg.Add(1)
	go w.flushLoop()

	return w
}

// Write 写入数据点
func (w *Writer) Write(point *write.Point) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, point)

	if len(w.buffer) >= w.batchSize {
		return w.flush()
	}

	return nil
}

// WriteBatch 批量写入数据点
func (w *Writer) WriteBatch(points ...*write.Point) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, points...)

	if len(w.buffer) >= w.batchSize {
		return w.flush()
	}

	return nil
}

// flush 刷新缓冲区（需要持有锁）
func (w *Writer) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	points := make([]*write.Point, len(w.buffer))
	copy(points, w.buffer)
	w.buffer = w.buffer[:0]

	// 增加 WaitGroup 计数，跟踪异步写入
	w.wg.Add(1)

	// 异步写入
	go func() {
		defer w.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := w.client.WriteBatchBlocking(ctx, points...)
		if err != nil {
			// 只在通道未关闭时发送错误
			w.mu.Lock()
			closed := w.closed
			w.mu.Unlock()

			if !closed {
				select {
				case w.errChan <- fmt.Errorf("batch write failed: %w", err):
				default:
					// 错误通道已满，丢弃错误
				}
			}
		}
	}()

	return nil
}

// Flush 强制刷新缓冲区
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}

// flushLoop 定时刷新循环
func (w *Writer) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			// 最后刷新一次
			w.Flush()
			return
		case <-ticker.C:
			w.Flush()
		}
	}
}

// Errors 获取错误通道
func (w *Writer) Errors() <-chan error {
	return w.errChan
}

// Close 关闭写入器
func (w *Writer) Close() error {
	// 先刷新缓冲区
	if err := w.Flush(); err != nil {
		return err
	}

	// 取消 context，停止刷新循环
	w.cancel()

	// 等待所有 goroutine 结束（包括 flushLoop 和所有异步写入）
	w.wg.Wait()

	// 标记为已关闭并关闭错误通道
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	close(w.errChan)

	return nil
}
