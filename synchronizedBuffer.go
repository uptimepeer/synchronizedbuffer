package synchronizedbuffer

import (
	"context"
	"sync"
	"time"
)

// Default settings for buffer size and flush time.
var DEFAULT_BUFFER_SIZE = 100
var DEFAULT_FLUSH_TIME = 1

// SynchronizedBuffer is a generic buffer that collects objects of any type T.
// It flushes either when the buffer is full or on a time interval, and supports
// graceful shutdown to ensure all data is processed before exit.
type SynchronizedBuffer[T any] struct {
	buffer         []T             // Buffer to store objects of type T
	mu             sync.Mutex      // Mutex to synchronize access to the buffer
	bufferFullChan chan struct{}   // Channel to signal when the buffer is full
	shutdownChan   chan struct{}   // Channel to signal a graceful shutdown
	wg             *sync.WaitGroup // WaitGroup to ensure all data is processed
	closeOnce      sync.Once       // Ensures shutdownChan is closed only once
	options        BufferOptions   // Buffer configuration options
}

// BufferOptions struct holds configurable settings for the buffer.
type BufferOptions struct {
	maxSize   int           // Maximum size of the buffer
	flushTime time.Duration // Interval duration for automatic flushes
}

// OptionFunc type defines a function for configuring buffer options.
type OptionFunc func(*BufferOptions)

// WithMaxSize sets the maximum buffer size.
func WithMaxSize(size int) OptionFunc {
	return func(o *BufferOptions) {
		o.maxSize = size
	}
}

// WithFlushTime sets the flush interval duration.
func WithFlushTime(duration time.Duration) OptionFunc {
	return func(o *BufferOptions) {
		o.flushTime = duration
	}
}

// New initializes a buffer with the specified flush function
// and options. It automatically flushes data based on the buffer's capacity or a time interval.
func New[T any](ctx context.Context, flushFunc func([]T), opts ...OptionFunc) *SynchronizedBuffer[T] {

	// Set default buffer options
	options := BufferOptions{
		maxSize:   DEFAULT_BUFFER_SIZE,
		flushTime: time.Duration(DEFAULT_FLUSH_TIME) * time.Second,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	// Use background context if no context provided
	if ctx == nil {
		ctx = context.Background()
	}

	// Initialize WaitGroup to track completion of flush operations
	var wg sync.WaitGroup
	wg.Add(1)

	// Initialize the buffer
	b := &SynchronizedBuffer[T]{
		buffer:         make([]T, 0, options.maxSize),
		bufferFullChan: make(chan struct{}, 1),
		shutdownChan:   make(chan struct{}),
		wg:             &wg,
		options:        options,
	}

	// Start a goroutine for periodic or on-demand flushing
	go b.startFlusher(ctx, flushFunc)

	return b
}

// Close signals a graceful shutdown of the buffer, flushing any remaining items.
// Ensures Close is only triggered once by using sync.Once.
func (b *SynchronizedBuffer[T]) Close() {
	b.closeOnce.Do(func() {
		close(b.shutdownChan)
	})
}

// Wait blocks until all buffered data has been flushed and processed.
func (b *SynchronizedBuffer[T]) Wait() {
	b.wg.Wait()
}

// Add adds an item to the buffer. When the buffer reaches its maximum size,
// it signals the need to flush via bufferFullChan.
func (b *SynchronizedBuffer[T]) Add(item T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Append item to the buffer
	b.buffer = append(b.buffer, item)

	// Signal if the buffer is full
	if len(b.buffer) >= b.options.maxSize {
		// Non-blocking send to bufferFullChan
		select {
		case b.bufferFullChan <- struct{}{}:
		default:
		}
	}
}

// Flush processes all items in the buffer and then clears it.
// Returns a slice of all items in the buffer to be passed to the flush function.
func (b *SynchronizedBuffer[T]) Flush() []T {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If the buffer is empty, return nil
	if len(b.buffer) == 0 {
		return nil
	}

	// Copy buffer contents and reset the buffer
	data := make([]T, len(b.buffer))
	copy(data, b.buffer)
	b.buffer = b.buffer[:0] // Clear buffer
	return data
}

// startFlusher continuously monitors for flush conditions and handles flushing
// when the buffer is full, on a timer, or during a graceful shutdown.
func (b *SynchronizedBuffer[T]) startFlusher(ctx context.Context, flushFunc func([]T)) {
	// Set up a ticker for regular interval flushes
	ticker := time.NewTicker(b.options.flushTime)
	defer ticker.Stop()
	defer b.wg.Done() // Mark work as done when this function exits

	for {
		select {
		case <-ctx.Done():
			// On context cancellation (e.g., app shutdown), flush remaining data
			if data := b.Flush(); data != nil {
				flushFunc(data)
			}
			return
		case <-b.shutdownChan:
			// On explicit Close call, flush remaining data
			if data := b.Flush(); data != nil {
				flushFunc(data)
			}
			return
		case <-ticker.C:
			// Flush buffer at regular intervals
			if data := b.Flush(); data != nil {
				flushFunc(data)
			}
		case <-b.bufferFullChan:
			// Flush buffer when it reaches capacity
			if data := b.Flush(); data != nil {
				flushFunc(data)
			}
		}
	}
}
