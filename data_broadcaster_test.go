package wesplot

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// testDataRowReader is a flexible DataRowReader for tests. It yields a
// sequence of DataRows or errors (via `items`), optionally sleeping between
// reads to simulate a live source.
type testDataRowReader struct {
	items []interface{} // each item is either DataRow or error
	delay time.Duration
	i     int
}

func newTestReaderFromRows(rows []DataRow, delay time.Duration) *testDataRowReader {
	items := make([]interface{}, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	return &testDataRowReader{items: items, delay: delay}
}

func newTestReaderFromItems(items []interface{}) *testDataRowReader {
	return &testDataRowReader{items: items}
}

func (r *testDataRowReader) Read(ctx context.Context) (DataRow, error) {
	if r.i >= len(r.items) {
		return DataRow{}, io.EOF
	}

	if r.delay > 0 {
		time.Sleep(r.delay)
	}

	v := r.items[r.i]
	r.i++

	switch vv := v.(type) {
	case DataRow:
		return vv, nil
	case error:
		return DataRow{}, vv
	default:
		return DataRow{}, fmt.Errorf("invalid seq item")
	}
}

func (r *testDataRowReader) ColumnNames() []string { return nil }

// collectFromChannels reads from one or more DataRow channels until each
// channel emits a row with `streamEnded == true`. It returns a slice of
// slices where each inner slice contains the rows (excluding the end
// marker) received on the corresponding input channel. If the provided
// timeout elapses before all channels finish, an error is returned.
func collectFromChannels(timeout time.Duration, chans ...<-chan DataRow) ([][]DataRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	n := len(chans)
	results := make([][]DataRow, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i, ch := range chans {
		i, ch := i, ch
		go func() {
			defer wg.Done()
			var local []DataRow
			for {
				select {
				case <-ctx.Done():
					results[i] = local
					return
				case r, ok := <-ch:
					if !ok || r.streamEnded {
						results[i] = local
						return
					}
					local = append(local, r)
				}
			}
		}()
	}

	wg.Wait()
	if err := ctx.Err(); err != nil {
		return results, fmt.Errorf("timeout waiting for channels: %v", err)
	}
	return results, nil
}

// extractDatas returns the payload `DataRowData` for each DataRow in the
// provided slice.
func extractDatas(rows []DataRow) []DataRowData {
	out := make([]DataRowData, len(rows))
	for i := range rows {
		out[i] = rows[i].DataRowData
	}
	return out
}

// recvRow reads a single DataRow from `ch` with a timeout and returns the
// DataRow and a boolean indicating success.
func recvRow(ch <-chan DataRow, timeout time.Duration) (DataRow, bool) {
	select {
	case r := <-ch:
		return r, true
	case <-time.After(timeout):
		return DataRow{}, false
	}
}

func TestDataBroadcaster(t *testing.T) {
	t.Run("ForwardingAndOrdering", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		// Register the channel before starting the broadcaster to test live
		// forwarding without relying on buffering.
		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, nil)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)

		d.Start(ctx)

		// Collect rows from the channel using helper
		res, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]

		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("data rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}

		d.Wait()
	})

	// RegisterSecondChannelAfterOneMessage: test registering a second channel
	// after one message has already been emitted (reader blocked for control).
	t.Run("RegisterSecondChannelAfterOneMessage", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)

		d.Start(ctx)

		// Register first channel; it should immediately receive the buffered
		// first row.
		ch1 := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch1)

		// Sends a message to ch1.
		br.Proceed()

		// This should time out as there is not EOF and only one message is sent
		// We ignore the error
		res, err := collectFromChannels(10*time.Millisecond, ch1)
		if err == nil {
			t.Fatalf("expected timeout collecting from ch1, but got result: %+v", res)
		}

		if len(res[0]) != 1 {
			t.Fatalf("expected one row on ch1 after first proceed, not %d", len(res[0]))
		}

		firstRowCh1 := res[0][0]

		if !reflect.DeepEqual(firstRowCh1.DataRowData, rows[0].DataRowData) {
			t.Fatalf("first row on ch1 mismatch: want %+v got %+v", rows[0].DataRowData, firstRowCh1.DataRowData)
		}

		// Register second channel after the first message has been emitted;
		// it should also receive the buffered first row upon registration.
		ch2 := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch2)

		// Also ignore the timeout as there's only one message sent.
		res, err = collectFromChannels(10*time.Millisecond, ch2)
		if err == nil {
			t.Fatalf("expected timeout collecting from ch1, but got result: %+v", res)
		}

		if len(res[0]) != 1 {
			t.Fatalf("expected one row on ch2 after registration, not %d", len(res[0]))
		}

		firstRowCh2 := res[0][0]

		if !reflect.DeepEqual(firstRowCh2.DataRowData, rows[0].DataRowData) {
			t.Fatalf("first row on ch2 mismatch: want %+v got %+v", rows[0].DataRowData, firstRowCh2.DataRowData)
		}

		// Now broadcast the remaining rows
		for i := 1; i < len(rows); i++ {
			br.Proceed()
		}

		// Collect from both channels until stream end
		res, err = collectFromChannels(2*time.Second, ch1, ch2)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		// We already consumed the first row above; prepend it so assertions
		// compare the full sequence.
		got1 := append([]DataRow{firstRowCh1}, res[0]...)
		got2 := append([]DataRow{firstRowCh2}, res[1]...)

		if !reflect.DeepEqual(extractDatas(got1), extractDatas(rows)) {
			t.Fatalf("ch1 full data mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got1))
		}
		if !reflect.DeepEqual(extractDatas(got2), extractDatas(rows)) {
			t.Fatalf("ch2 full data mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got2))
		}

		d.Wait()
	})

	t.Run("DeregisterSingleChannel", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		// allow first row
		br.Proceed()
		first, ok := recvRow(ch, 200*time.Millisecond)
		if !ok {
			t.Fatalf("expected first row for single channel, got none")
		}
		if !reflect.DeepEqual(first.DataRowData, rows[0].DataRowData) {
			t.Fatalf("first row mismatch: want %+v got %+v", rows[0].DataRowData, first.DataRowData)
		}

		// deregister the only channel
		d.DeregisterChannel(ctx, ch)

		// allow remaining rows to be emitted
		for i := 1; i < len(rows); i++ {
			br.Proceed()
		}

		d.Wait()

		// ensure no more rows were sent to the deregistered channel
		select {
		case r := <-ch:
			t.Fatalf("received unexpected row on deregistered single channel: %+v", r)
		default:
			// ok
		}
	})

	t.Run("DeregisterAmongMultipleChannels", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)

		ch1 := make(chan DataRow, 10)
		ch2 := make(chan DataRow, 10)
		ch3 := make(chan DataRow, 10)
		ch4 := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch1)
		d.RegisterChannel(ctx, ch2)
		d.RegisterChannel(ctx, ch3)
		d.RegisterChannel(ctx, ch4)
		d.Start(ctx)

		// allow first row
		br.Proceed()
		f1, ok1 := recvRow(ch1, 200*time.Millisecond)
		f2, ok2 := recvRow(ch2, 200*time.Millisecond)
		f3, ok3 := recvRow(ch3, 200*time.Millisecond)
		f4, ok4 := recvRow(ch4, 200*time.Millisecond)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			t.Fatalf("expected first row on all channels, got ok1=%v ok2=%v ok3=%v ok4=%v", ok1, ok2, ok3, ok4)
		}
		if !reflect.DeepEqual(f1.DataRowData, rows[0].DataRowData) || !reflect.DeepEqual(f2.DataRowData, rows[0].DataRowData) || !reflect.DeepEqual(f3.DataRowData, rows[0].DataRowData) || !reflect.DeepEqual(f4.DataRowData, rows[0].DataRowData) {
			t.Fatalf("first row mismatch on channels: ch1=%+v ch2=%+v ch3=%+v ch4=%+v", f1.DataRowData, f2.DataRowData, f3.DataRowData, f4.DataRowData)
		}

		// deregister ch1 and ch3
		d.DeregisterChannel(ctx, ch1)
		d.DeregisterChannel(ctx, ch3)

		// allow remaining rows
		for i := 1; i < len(rows); i++ {
			br.Proceed()
		}

		// collect from ch2 and ch4 until stream end
		res, err := collectFromChannels(2*time.Second, ch2, ch4)
		if err != nil {
			t.Fatalf("collectFromChannels failed for remaining channels: %v", err)
		}
		got2 := append([]DataRow{f2}, res[0]...)
		got4 := append([]DataRow{f4}, res[1]...)
		if !reflect.DeepEqual(extractDatas(got2), extractDatas(rows)) {
			t.Fatalf("ch2 did not receive full data after others deregistered: want %+v got %+v", extractDatas(rows), extractDatas(got2))
		}
		if !reflect.DeepEqual(extractDatas(got4), extractDatas(rows)) {
			t.Fatalf("ch4 did not receive full data after others deregistered: want %+v got %+v", extractDatas(rows), extractDatas(got4))
		}

		// ensure ch1 and ch3 received no more than the first row
		select {
		case r := <-ch1:
			t.Fatalf("received unexpected row on deregistered ch1: %+v", r)
		default:
			// ok
		}
		select {
		case r := <-ch3:
			t.Fatalf("received unexpected row on deregistered ch3: %+v", r)
		default:
			// ok
		}

		d.Wait()
	})

	// Error-handling subtests: ensure errIgnoreThisRow is skipped and underlying
	// reader errors are propagated via the stream-end marker.
	t.Run("IgnoreThisRowIsSkipped", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		// Sequence: first an ignore error, then the three rows, then EOF
		items := []interface{}{errIgnoreThisRow, rows[0], rows[1], rows[2]}
		r := newTestReaderFromItems(items)
		d := NewDataBroadcaster(r, 10, nil)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		res, err := collectFromChannels(1*time.Second, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]
		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("unexpected rows after ignore: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}

		d.Wait()
	})

	t.Run("UnderlyingErrorEndsStreamWithError", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{100}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{200}}},
		}

		boom := fmt.Errorf("boom error")
		// Sequence: two normal rows then an underlying error
		items := []interface{}{rows[0], rows[1], boom}
		r := newTestReaderFromItems(items)
		d := NewDataBroadcaster(r, 10, nil)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		// Read until streamEnded marker is seen and capture rows and final error
		var received []DataRow
		var finalErr error
		timeout := time.After(2 * time.Second)
	loop:
		for {
			select {
			case <-timeout:
				t.Fatalf("timeout waiting for end marker")
			case rcv := <-ch:
				if rcv.streamEnded {
					finalErr = rcv.streamErr
					break loop
				}
				received = append(received, rcv)
			}
		}

		if !reflect.DeepEqual(extractDatas(received), extractDatas(rows)) {
			t.Fatalf("received rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(received))
		}
		if finalErr == nil || finalErr.Error() != boom.Error() {
			t.Fatalf("expected final error %v, got %v", boom, finalErr)
		}

		d.Wait()
	})

	// LateRegisterWhenBufferCapacityExceeded: small buffer overflow case.
	t.Run("LateRegisterWhenBufferCapacityExceeded", func(t *testing.T) {
		ctx := context.Background()

		// Buffer capacity 2, emit 4 rows; only the last 2 should be buffered.
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{1}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{2}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{3}}},
			{DataRowData: DataRowData{X: 4, Ys: []float64{4}}},
		}

		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 2, nil)
		d.Start(ctx)
		d.Wait()

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)

		tmp, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		res := tmp[0]

		// Because the DataBroadcaster appends a stream-end marker into the ring
		// at the end, that consumes one slot. Therefore after stream end the
		// ring will only contain (capacity-1) data rows; adjust expectation
		// accordingly for capacity=2 => expect last 1 row.
		want := rows[len(rows)-1:]
		if !reflect.DeepEqual(extractDatas(res), extractDatas(want)) {
			t.Fatalf("buffered rows mismatch after overflow: want %+v got %+v", extractDatas(want), extractDatas(res))
		}
	})

	// LateRegisterAfterMultipleBufferRotations: many rows, expect last N preserved.
	t.Run("LateRegisterAfterMultipleBufferRotations", func(t *testing.T) {
		ctx := context.Background()

		// Buffer 3, emit 10 rows; expect last 3 to be buffered.
		var rows []DataRow
		for i := 1; i <= 10; i++ {
			rows = append(rows, DataRow{DataRowData: DataRowData{X: float64(i), Ys: []float64{float64(i)}}})
		}

		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 3, nil)
		d.Start(ctx)
		d.Wait()

		ch := make(chan DataRow, 20)
		d.RegisterChannel(ctx, ch)

		tmp, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		res := tmp[0]
		// For capacity=3 we expect the last (3-1)=2 data rows to remain after
		// the final stream-end marker has been pushed.
		want := rows[len(rows)-2:]
		if !reflect.DeepEqual(extractDatas(res), extractDatas(want)) {
			t.Fatalf("buffered rows mismatch after many rotations: want %+v got %+v", extractDatas(want), extractDatas(res))
		}
	})

	t.Run("BufferingLateRegister", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
		}

		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, nil)
		d.Start(ctx)

		// Wait for broadcaster to finish emitting all rows.
		d.Wait()

		// Register a new channel after the stream ended; it should receive the
		// buffered data. Use helper to collect buffered rows (excludes end marker).
		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)

		res, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]

		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("buffered rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}
	})

	t.Run("MultipleChannelsReceiveSameData", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
		}

		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, nil)

		ch1 := make(chan DataRow, 10)
		ch2 := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch1)
		d.RegisterChannel(ctx, ch2)

		d.Start(ctx)

		res, err := collectFromChannels(1*time.Second, ch1, ch2)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got1 := res[0]
		got2 := res[1]

		if !reflect.DeepEqual(extractDatas(got1), extractDatas(got2)) {
			t.Fatalf("channels differ: ch1=%+v ch2=%+v", extractDatas(got1), extractDatas(got2))
		}
		if !reflect.DeepEqual(extractDatas(got1), extractDatas(rows)) {
			t.Fatalf("channel data mismatch vs source rows: want %+v got %+v", extractDatas(rows), extractDatas(got1))
		}

		d.Wait()
	})

	t.Run("TeeOutputDisabled", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.5, Ys: []float64{10.25}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{20.5, 30.75}}},
		}

		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, nil)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		res, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]

		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("data rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}

		d.Wait()
	})

	t.Run("TeeOutputEnabled", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.5, Ys: []float64{10.25}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{20.5, 30.75}}},
		}

		var buf strings.Builder
		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, &buf)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		res, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]

		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("data rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}

		d.Wait()

		teeOutput := buf.String()
		expectedLines := []string{
			"1.500000,10.250000",
			"2.000000,20.500000,30.750000",
		}

		lines := strings.Split(strings.TrimSpace(teeOutput), "\n")
		if len(lines) != len(expectedLines) {
			t.Fatalf("expected %d tee output lines, got %d: %q", len(expectedLines), len(lines), teeOutput)
		}

		for i, expected := range expectedLines {
			if lines[i] != expected {
				t.Errorf("tee output line %d mismatch: want %q got %q", i, expected, lines[i])
			}
		}
	})

	t.Run("TeeOutputWithMultipleYValues", func(t *testing.T) {
		ctx := context.Background()

		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0, 20.0, 30.0, 40.0}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{15.0, 25.0, 35.0, 45.0}}},
			{DataRowData: DataRowData{X: 3.0, Ys: []float64{12.5, 22.5, 32.5, 42.5}}},
		}

		var buf strings.Builder
		reader := newTestReaderFromRows(rows, 1*time.Millisecond)
		d := NewDataBroadcaster(reader, 10, &buf)

		ch := make(chan DataRow, 10)
		d.RegisterChannel(ctx, ch)
		d.Start(ctx)

		res, err := collectFromChannels(500*time.Millisecond, ch)
		if err != nil {
			t.Fatalf("collectFromChannels failed: %v", err)
		}
		got := res[0]

		if !reflect.DeepEqual(extractDatas(got), extractDatas(rows)) {
			t.Fatalf("data rows mismatch: want %+v got %+v", extractDatas(rows), extractDatas(got))
		}

		d.Wait()

		teeOutput := buf.String()
		expectedLines := []string{
			"1.000000,10.000000,20.000000,30.000000,40.000000",
			"2.000000,15.000000,25.000000,35.000000,45.000000",
			"3.000000,12.500000,22.500000,32.500000,42.500000",
		}

		lines := strings.Split(strings.TrimSpace(teeOutput), "\n")
		if len(lines) != len(expectedLines) {
			t.Fatalf("expected %d tee output lines, got %d: %q", len(expectedLines), len(lines), teeOutput)
		}

		for i, expected := range expectedLines {
			if lines[i] != expected {
				t.Errorf("tee output line %d mismatch: want %q got %q", i, expected, lines[i])
			}
		}
	})
}

// blockingDataRowReader yields the first row immediately and then blocks on
// a channel until the test tells it to continue. Each call to `Proceed`
// allows the reader to return exactly one more row. This is used to
// deterministically control when the DataBroadcaster reads the next value.
type blockingDataRowReader struct {
	rows    []DataRow
	i       int
	proceed chan struct{}
}

func (b *blockingDataRowReader) Read(ctx context.Context) (DataRow, error) {
	if b.i >= len(b.rows) {
		return DataRow{}, io.EOF
	}

	// Wait for a proceed signal from the test before returning any row.
	select {
	case <-b.proceed:
	case <-ctx.Done():
		return DataRow{}, ctx.Err()
	}

	r := b.rows[b.i]
	b.i++
	return r, nil
}

func (b *blockingDataRowReader) ColumnNames() []string { return nil }

// Proceed unblocks one pending reader call so it can return the next row.
func (b *blockingDataRowReader) Proceed() {
	b.proceed <- struct{}{}
}
