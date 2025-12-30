package wesplot

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/trace"
	"strings"
	"sync"
	"sync/atomic"
)

type DataBroadcaster struct {
	// The data row reader to be read from.
	input DataRowReader

	teeOutput io.Writer

	mutex sync.Mutex
	wg    sync.WaitGroup

	// If the stream is ended or not
	streamEnded atomic.Bool
	err         error // The error emited by the Run(), if any. Should be read after streamEnded == true to ensure no data race.

	// These are channels from open websockets where we are sending data to.
	// Channels should be buffered, to not block the DataBroadcaster.
	channelsForLiveUpdate []chan<- DataRow

	// This contains the most recent data received. The data in this ring will be
	// sent to channel upon registration. See RegisterChannel for details.
	//
	// TODO: potentially switch to an allocating, time-based ring buffer instead
	// of this.
	dataBuffer *ThreadUnsafeRing[DataRow]

	// Just for tracking how many rows are emitted when EOF is encountered.
	numDataRowsEmitted int

	logger *slog.Logger
}

func NewDataBroadcaster(input DataRowReader, bufferCapacity int, teeOutput io.Writer) *DataBroadcaster {
	return &DataBroadcaster{
		input: input,

		teeOutput: teeOutput,

		mutex:                 sync.Mutex{},
		channelsForLiveUpdate: make([]chan<- DataRow, 0),
		dataBuffer:            NewRing[DataRow](bufferCapacity),
		numDataRowsEmitted:    0,
		logger:                slog.Default().With("tag", "DataBroadcaster"),
	}
}

func (d *DataBroadcaster) Start(ctx context.Context) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := d.run(ctx)

		d.err = err

		// Must set all variables to be read after DataBroadcaster is complete before
		// this, as this atomic is used to "release" all the other variables (see Golang
		// memory model)
		d.streamEnded.Store(true)

		// Maybe we should deregister and close the currently registered channels.
		// However, to do this safely, more analysis is needed on the concurrency
		// and see if there are bugs.
		// Also caching the end message in the cache will allow newly connected
		// clients to know this easier.
		// TODO: check the above.
		d.cacheAndBroadcastData(ctx, DataRow{
			streamEnded: true,
			streamErr:   err,
		})

		logger := d.logger.With("numDataRowsEmitted", d.numDataRowsEmitted)
		if err != nil {
			logger = logger.With("error", err)
		}
		logger.Info("data broadcaster stream ended")
	}()
}

func (d *DataBroadcaster) Wait() {
	d.wg.Wait()
}

// Register a new channel. Called from the HTTP server when a new websocket
// connection is initiated.
//
// - ctx: is the HTTP call context.
// - c: is the channel to send data on. This should be a buffered channel to ensure the DataBroadcaster is not blocked, as if any channel is blocked, everything is blocked.
func (d *DataBroadcaster) RegisterChannel(ctx context.Context, c chan<- DataRow) {
	// Note: this method should only be called by the HTTP server thread and not
	// the DataBroadcaster thread.
	//
	// We have to take a "global" lock (well, there's only a single
	// DataBroadcaster goroutine per process) because we want to push all buffered
	// data to the client. After the buffered data is pushed to the client, we
	// have to ensure no subsequent data is missed by the client due to race
	// conditions (serialization and deserialization of the data can be time
	// consuming).
	//
	// To accomplish this, whenever we register a new channel (i.e. a new browser
	// client opens against this process), we take a global mutex on the
	// DataBroadcaster. The mutex will only be locked when no data is being sent
	// to the existing channels for live update. While the mutex is locked, no
	// additional data can be written to the buffers nor sent down the existing
	// channels. At this time, this code will then send all the buffered data to
	// the newly registered channels and add the channels into the list of
	// channels for live update. Only then it will unlock, which allows the main
	// DataBroadcaster to continue. Once continued, it will add the next message
	// into the cache and also it to all the channels, which will now include the
	// newly registered channels. This ensures no messages are missed in this
	// pipeline.
	//
	// This simple implementation means there can be a small amount of latency
	// when adding channels which may result in the "lock up" of all real-time
	// plots. This trade off is accepted as adding a new client (basically a new
	// tab) is not common. The latency is logged and can be measured via pprof.

	traceCtx, task := trace.NewTask(ctx, "RegisterChannel")
	defer task.End()

	trace.WithRegion(traceCtx, "Lock", d.mutex.Lock)
	defer d.mutex.Unlock()

	// First, we push all the buffered data to this channel to make sure it has all the histories.
	trace.WithRegion(traceCtx, "pushBufferedDataToChannel", func() {
		d.pushBufferedDataToChannel(c)
	})

	// Second, we add the channel into the list of channels we want to live update.
	// Not tracing this because it should be insignificant in terms of time taken
	d.channelsForLiveUpdate = append(d.channelsForLiveUpdate, c)

	d.logger.With(
		"newChannel", c,
		"channels", d.channelsForLiveUpdate,
	).Info("registered channel")
}

// Deregister a channel to get data updates. Called when a websocket client
// disconnects or when the input stream closes. Note: the channel shouldn't be
// closed until this method returns (if the input is still open), as it may
// cause panics otherwise.
//
// - ctx: is the HTTP call context.
// - c: is the channel to send data on. This should be the same channel as the one passed to RegisterChannel to successfully deregister.
//
// This method will panic if c is not registered. This indicates a programming error.
func (d *DataBroadcaster) DeregisterChannel(ctx context.Context, c chan<- DataRow) {
	traceCtx, task := trace.NewTask(ctx, "DeregisterChannel")
	defer task.End()

	trace.WithRegion(traceCtx, "Lock", d.mutex.Lock)
	defer d.mutex.Unlock()

	d.channelsForLiveUpdate = Filter(d.channelsForLiveUpdate, func(channel chan<- DataRow) bool {
		return channel != c
	})
	d.logger.With(
		"removedChannel", c,
		"channels", d.channelsForLiveUpdate,
	).Info("deregistered channel")
}

func (d *DataBroadcaster) run(ctx context.Context) error {
	var dataRow DataRow
	var err error

	for {
		traceCtx, task := trace.NewTask(ctx, "DataBroadcasterLoop")

		trace.WithRegion(traceCtx, "DataSourceRead", func() {
			dataRow, err = d.input.Read(traceCtx)
		})

		if err == errIgnoreThisRow {
			task.End()
			continue
		} else if err == io.EOF {
			// The source has ended. We don't want to close the channel or anything
			// like that, because we want to display the cached data and new browser
			// tabs could come online still.
			task.End()
			return nil
		} else if err != nil {
			task.End()
			return err
		}

		if d.teeOutput != nil {
			// Kind of inefficient, but probably OK.
			dataLine := make([]string, 0, len(dataRow.Ys)+1)
			dataLine = append(dataLine, fmt.Sprintf("%f", dataRow.X))

			for _, y := range dataRow.Ys {
				dataLine = append(dataLine, fmt.Sprintf("%f", y))
			}

			fmt.Fprintln(d.teeOutput, strings.Join(dataLine, ","))
		}

		d.cacheAndBroadcastData(traceCtx, dataRow)
		task.End()
	}
}

func (d *DataBroadcaster) cacheAndBroadcastData(traceCtx context.Context, dataRow DataRow) {
	d.numDataRowsEmitted++

	trace.WithRegion(traceCtx, "Lock", d.mutex.Lock)
	defer d.mutex.Unlock()

	d.logger.With(
		"x", dataRow.X,
		"ys", dataRow.Ys,
	).Debug("new data row")

	trace.WithRegion(traceCtx, "Cache", func() {
		d.dataBuffer.Push(dataRow)
	})

	trace.WithRegion(traceCtx, "Broadcast", func() {
		for _, c := range d.channelsForLiveUpdate {
			c <- dataRow
		}
	})
}

func (d *DataBroadcaster) pushBufferedDataToChannel(c chan<- DataRow) {
	bufferedData := d.dataBuffer.ReadAllOrdered()

	for _, dataRow := range bufferedData {
		c <- dataRow
	}
}
