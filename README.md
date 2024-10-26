# SynchronizedBuffer

`SynchronizedBuffer` is a generic, thread-safe Go library for managing buffered storage of any data type. It flushes data to a given function either when the buffer reaches a specified size or at regular time intervals. The library includes a graceful shutdown mechanism, ensuring that all buffered data is processed before termination.

## Features

- **Generic Buffer**: Supports buffering of any data type.
- **Customizable Options**: Configure buffer size and flush interval.
- **Thread-Safe**: Uses synchronization to safely handle concurrent writes.
- **Graceful Shutdown**: Flushes all remaining data before shutdown.

## Installation

To install the library, use:

```bash
go get -u github.com/uptimepeer/synchronizedbuffer
```

## Usage

### Importing the Library

```go
import "github.com/uptimepeer/synchronizedbuffer"
```

### Basic Example

The following example demonstrates how to initialize a `SynchronizedBuffer`, add data to it, and trigger flushes based on buffer capacity or time intervals.

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/uptimepeer/synchronizedbuffer"
)

// Define a sample struct type for buffer data
type SampleStruct struct {
    ID          uint
    Name        string
}

func main() {
    // Define a context to manage the buffer lifecycle
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel() // Ensure that the cancel function is called to free resources 
    
    // Define a flush function to handle batch processing
    flushFunc := func(data []SampleStruct) {
        fmt.Printf("Flushing %d items to the database...\n", len(data))
        for _, item := range data {
            fmt.Printf("Saving item: %+v\n", item)
        }
    }

    // Initialize the SynchronizedBuffer with a max size and flush interval
    buffer := synchronizedbuffer.New(ctx, flushFunc,
        synchronizedbuffer.WithMaxSize(10),                  // Max buffer size of 10
        synchronizedbuffer.WithFlushTime(5*time.Second))     // Flush every 5 seconds

    // Simulate adding data to the buffer
    for i := 0; i < 25; i++ {
        buffer.Add(SampleStruct{
            ID:   uint(i),
            Name: fmt.Sprintf("Name-%d", i),
        })
        time.Sleep(500 * time.Millisecond) // Simulate delay between writes
    }

    // Close and wait to gracefully flush remaining data
    buffer.Close()
    buffer.Wait()
	fmt.Println("All data flushed.")
}
```

> **Note**: The buffer can be shut down either by calling the `Close()` function or by canceling the provided context. If no context operation is required, it is safe to pass `nil` for the context when creating a new buffer.

### Graceful Shutdown

To trigger a graceful shutdown of the buffer, you can call `Close()`. This will stop the flushing goroutine and flush any remaining data in the buffer. Use `Wait()` to block until all data has been flushed.

```go
// Close the buffer to initiate shutdown
buffer.Close()

// Wait for the buffer to finish processing
buffer.Wait()
```

> **Note**: If the buffer is closed by canceling the context (using the `cancel()` function), you do not need to call `Close()` again. 

### Configuration Options

You can customize the behavior of `SynchronizedBuffer` by passing options when initializing:


- **WithMaxSize(int)**: Set the maximum buffer capacity. When this limit is reached, the buffer flushes automatically.
- **WithFlushTime(time.Duration)**: Set a time interval for automatic flushes, even if the buffer is not full.

> **Note**: If options are not provided, default value for `MaxSize` is 100 and defult value for `FlushTime` is 1 second.

### Example with Options

```go
buffer := synchronizedbuffer.New(ctx, flushFunc,
    synchronizedbuffer.WithMaxSize(20),                // Set max buffer size to 20 items
    synchronizedbuffer.WithFlushTime(3*time.Second))   // Set flush interval to every 3 seconds
```

## API Reference

### New

```go
func New[T any](
    ctx context.Context,
    flushFunc func([]T),
    opts ...OptionFunc,
) *SynchronizedBuffer[T]
```

- **ctx**: Context to manage buffer lifecycle.
- **flushFunc**: Function to process a batch of buffered items.
- **opts**: Variadic parameter to specify buffer options (size, flush interval).

### Add

```go
func (b *SynchronizedBuffer[T]) Add(item T)
```

Adds an item to the buffer. If the buffer is full, it will automatically trigger a flush.

### Flush

```go
func (b *SynchronizedBuffer[T]) Flush() []T
```

Manually flushes the buffer and returns the flushed items. This can be used any time to immediately process the buffer’s current contents.


### Close

```go
func (b *SynchronizedBuffer[T]) Close()
```

Signals a graceful shutdown, flushing all remaining data and stopping the flushing goroutine.

### Wait

```go
func (b *SynchronizedBuffer[T]) Wait()
```

Blocks until all buffered data has been processed. Call after `Close()` to ensure all items have been flushed.

## Contributing

Feel free to open issues or submit pull requests with improvements or bug fixes.

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
