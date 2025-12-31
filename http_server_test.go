package wesplot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func startTestServer(metadata Metadata, broadcaster *DataBroadcaster) (string, func()) {
	// Use NewHttpServer to ensure the same handler registration and behavior
	// as production code. We deliberately do not call `Run()` to avoid
	// side-effects such as opening a browser or binding to a specific port.
	s := NewHttpServer(broadcaster, "127.0.0.1", 0, metadata, 10*time.Millisecond)

	srv := httptest.NewServer(s.mux)

	cleanup := func() {
		srv.Close()
		if broadcaster != nil {
			broadcaster.Wait()
		}
	}

	return srv.URL, cleanup
}

// fetchMetadata performs a GET against /metadata on the provided baseURL,
// decodes the JSON response into Metadata and returns the response and any
// error encountered.
func fetchMetadata(baseURL string) (Metadata, *http.Response, error) {
	var m Metadata
	resp, err := http.Get(baseURL + "/metadata")
	if err != nil {
		return m, nil, err
	}

	// Attempt to decode the body. Note: callers are responsible for closing
	// resp.Body when finished (we close it on decoding error to avoid leaks).
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		resp.Body.Close()
		return m, resp, err
	}

	return m, resp, nil
}

// fetchErrors performs a GET against /errors on the provided baseURL,
// decodes the JSON response into the typed StreamEndedMessage and returns
// the response and any error encountered. This helper does not perform
// assertions so callers can assert headers/status as needed.
func fetchErrors(baseURL string) (StreamEndedMessage, *http.Response, error) {
	var res StreamEndedMessage

	resp, err := http.Get(baseURL + "/errors")
	if err != nil {
		return StreamEndedMessage{}, nil, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		resp.Body.Close()
		return StreamEndedMessage{}, resp, err
	}

	return res, resp, nil
}

// dialWebSocket opens a websocket connection to the /ws endpoint for tests.
// Caller is responsible for closing the returned cleanup function.
func dialWebSocket(baseURL string) (*websocket.Conn, func(), error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse baseURL: %w", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("dial websocket: %w", err)
	}

	cleanup := func() {
		c.Close(websocket.StatusNormalClosure, "")
	}

	return c, cleanup, nil
}

// readWebsocketRows reads the next message as []DataRow with a timeout.
func readWebsocketRows(c *websocket.Conn, timeout time.Duration) ([]DataRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var rows []DataRow
	if err := wsjson.Read(ctx, c, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

// waitWebsocketClosed waits for a normal websocket closure; tolerates empty flushes before closing.
func waitWebsocketClosed(c *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	for {
		var rows []DataRow
		err := wsjson.Read(ctx, c, &rows)
		if err != nil {
			if status := websocket.CloseStatus(err); status != websocket.StatusNormalClosure {
				return fmt.Errorf("unexpected websocket close status: %v", status)
			}
			return nil
		}

		// The server may flush an empty buffer before closing; ignore it.
		if len(rows) == 0 {
			continue
		}

		return fmt.Errorf("expected websocket to close, got data instead: %+v", rows)
	}
}

func TestHTTPServer_Metadata(t *testing.T) {
	// Subtest: ensure the endpoint returns the expected metadata JSON
	t.Run("ReturnsExpectedMetadata", func(t *testing.T) {
		expected := Metadata{
			WindowSize:    123,
			XIsTimestamp:  true,
			RelativeStart: false,
			WesplotOptions: WesplotOptions{
				Title:     "test title",
				Columns:   []string{"a", "b"},
				XLabel:    "x",
				YLabel:    "y",
				YUnit:     "u",
				ChartType: "line",
			},
		}

		baseURL, cleanup := startTestServer(expected, nil)
		defer cleanup()

		got, resp, err := fetchMetadata(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("unexpected Content-Type: %q", ct)
		}

		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
		}

		if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "content-type" {
			t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
		}

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("metadata mismatch:\nwant: %+v\ngot:  %+v", expected, got)
		}
	})

	// Subtest: CORS headers on metadata
	t.Run("CORSHeaders", func(t *testing.T) {
		baseURL, cleanup := startTestServer(Metadata{}, nil)
		defer cleanup()

		resp, err := http.Get(baseURL + "/metadata")
		if err != nil {
			t.Fatalf("failed to GET /metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "content-type" {
			t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
		}
	})

	// Subtest: YMin and YMax are nil (should round-trip as nil)
	t.Run("NilYMinYMax", func(t *testing.T) {
		expected := Metadata{
			WindowSize:    42,
			XIsTimestamp:  false,
			RelativeStart: false,
			WesplotOptions: WesplotOptions{
				Title:     "nil bounds",
				Columns:   []string{"a"},
				XLabel:    "x",
				YLabel:    "y",
				YUnit:     "u",
				ChartType: "line",
				YMin:      nil,
				YMax:      nil,
			},
		}

		baseURL, cleanup := startTestServer(expected, nil)
		defer cleanup()

		got, resp, err := fetchMetadata(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("metadata mismatch for nil bounds:\nwant: %+v\ngot:  %+v", expected, got)
		}
	})

	// Subtest: YMin and YMax are non-nil (should round-trip with values)
	t.Run("NonNilYMinYMax", func(t *testing.T) {
		ymin := 1.23
		ymax := 4.56
		expected := Metadata{
			WindowSize:    7,
			XIsTimestamp:  true,
			RelativeStart: true,
			WesplotOptions: WesplotOptions{
				Title:     "bounds",
				Columns:   []string{"a", "b"},
				XLabel:    "x",
				YLabel:    "y",
				YUnit:     "u",
				ChartType: "bar",
				YMin:      &ymin,
				YMax:      &ymax,
			},
		}

		baseURL, cleanup := startTestServer(expected, nil)
		defer cleanup()

		got, resp, err := fetchMetadata(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("metadata mismatch for bounds:\nwant: %+v\ngot:  %+v", expected, got)
		}
	})
}

func TestHTTPServer_Errors(t *testing.T) {
	// Subtest: stream ended without error
	t.Run("NoError", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{{DataRowData: DataRowData{X: 1, Ys: []float64{10}}}}
		r := newTestReaderFromRows(rows, 0)
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)
		d.Wait()

		baseURL, cleanup := startTestServer(Metadata{}, d)
		defer cleanup()

		res, resp, err := fetchErrors(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch /errors: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if !res.StreamEnded {
			t.Fatalf("expected StreamEnded true")
		}

		// StreamError should be empty when there is no error
		if res.StreamError != "" {
			t.Fatalf("expected StreamError to be empty when no error, got: %q", res.StreamError)
		}
	})

	t.Run("NotEndedAndNoErrors", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{{DataRowData: DataRowData{X: 1, Ys: []float64{10}}}}
		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(Metadata{}, d)

		// Do NOT finish the reader yet; the broadcaster should be running and not ended.
		res, resp, err := fetchErrors(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch /errors: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if res.StreamEnded {
			t.Fatalf("expected StreamEnded false while stream is running")
		}
		if res.StreamError != "" {
			t.Fatalf("expected empty StreamError while stream is running, got: %q", res.StreamError)
		}

		// Finish the reader so cleanup can wait for broadcaster to finish.
		br.Proceed()
		cleanup()
	})

	t.Run("WithError", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{{DataRowData: DataRowData{X: 1, Ys: []float64{10}}}}
		boom := fmt.Errorf("boom error")
		items := []interface{}{rows[0], boom}
		r := newTestReaderFromItems(items)
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)
		d.Wait()

		baseURL, cleanup := startTestServer(Metadata{}, d)
		defer cleanup()

		res, resp, err := fetchErrors(baseURL)
		if err != nil {
			t.Fatalf("failed to fetch /errors: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if !res.StreamEnded {
			t.Fatalf("expected StreamEnded true")
		}

		// StreamError should not be empty when the source errored
		if res.StreamError == "" {
			t.Fatalf("expected StreamError to be non-empty when an error occurred")
		}

		// Also assert the message content is present
		if !strings.Contains(res.StreamError, "boom error") {
			t.Fatalf("expected StreamError message to contain %q, got %q", "boom error", res.StreamError)
		}
	})

	t.Run("CORSHeaders", func(t *testing.T) {
		// Use a non-nil broadcaster (not started) so handler can access it safely.
		d := NewDataBroadcaster(newTestReaderFromRows([]DataRow{}, 0), 10, nil)
		baseURL, cleanup := startTestServer(Metadata{}, d)
		defer cleanup()

		resp, err := http.Get(baseURL + "/errors")
		if err != nil {
			t.Fatalf("failed to GET /errors: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, http.StatusOK)
		}

		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "content-type" {
			t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
		}
		if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "*" {
			t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
		}
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("unexpected Content-Type: %q", ct)
		}
	})
}

func TestHTTPServer_WebSocket(t *testing.T) {
	t.Run("SingleConnectionReceivesData", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
		}

		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(Metadata{WindowSize: 10}, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		br.Proceed()
		msg, err := readWebsocketRows(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}
		if len(msg) != 1 {
			t.Fatalf("expected 1 row in first websocket message, got %d", len(msg))
		}
		if !reflect.DeepEqual(msg[0].DataRowData, rows[0].DataRowData) {
			t.Fatalf("first websocket row mismatch: want %+v got %+v", rows[0].DataRowData, msg[0].DataRowData)
		}

		br.Proceed()
		msg, err = readWebsocketRows(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}
		if len(msg) != 1 {
			t.Fatalf("expected 1 row in second websocket message, got %d", len(msg))
		}
		if !reflect.DeepEqual(msg[0].DataRowData, rows[1].DataRowData) {
			t.Fatalf("second websocket row mismatch: want %+v got %+v", rows[1].DataRowData, msg[0].DataRowData)
		}

		if err := waitWebsocketClosed(c); err != nil {
			t.Fatalf("wait websocket close: %v", err)
		}
	})

	t.Run("SecondConnectionReceivesBufferedData", func(t *testing.T) {
		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1, Ys: []float64{10}}},
			{DataRowData: DataRowData{X: 2, Ys: []float64{20}}},
			{DataRowData: DataRowData{X: 3, Ys: []float64{30}}},
			{DataRowData: DataRowData{X: 4, Ys: []float64{40}}},
		}

		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(Metadata{WindowSize: 10}, d)
		defer cleanup()

		c1, closeC1, err := dialWebSocket(baseURL)
		if err != nil {
			t.Fatalf("dial websocket c1: %v", err)
		}
		defer closeC1()

		// Send first row to first connection only.
		br.Proceed()
		msg1, err := readWebsocketRows(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c1: %v", err)
		}
		if len(msg1) != 1 {
			t.Fatalf("expected 1 row in first message for c1, got %d", len(msg1))
		}
		if !reflect.DeepEqual(msg1[0].DataRowData, rows[0].DataRowData) {
			t.Fatalf("c1 first row mismatch: want %+v got %+v", rows[0].DataRowData, msg1[0].DataRowData)
		}

		// Now a second client connects; it should receive the buffered first row immediately.
		c2, closeC2, err := dialWebSocket(baseURL)
		if err != nil {
			t.Fatalf("dial websocket c2: %v", err)
		}
		defer closeC2()
		msg2, err := readWebsocketRows(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c2: %v", err)
		}
		if len(msg2) != 1 {
			t.Fatalf("expected 1 buffered row for c2, got %d", len(msg2))
		}
		if !reflect.DeepEqual(msg2[0].DataRowData, rows[0].DataRowData) {
			t.Fatalf("c2 buffered row mismatch: want %+v got %+v", rows[0].DataRowData, msg2[0].DataRowData)
		}

		// Deliver the remaining rows and expect both clients to receive them in order.
		br.Proceed()
		msg1, err = readWebsocketRows(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c1 second: %v", err)
		}
		msg2, err = readWebsocketRows(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c2 second: %v", err)
		}
		if len(msg1) != 1 || len(msg2) != 1 {
			t.Fatalf("expected 1 row for each client on second message, got len(msg1)=%d len(msg2)=%d", len(msg1), len(msg2))
		}
		if !reflect.DeepEqual(msg1[0].DataRowData, rows[1].DataRowData) || !reflect.DeepEqual(msg2[0].DataRowData, rows[1].DataRowData) {
			t.Fatalf("second row mismatch: c1=%+v c2=%+v want %+v", msg1[0].DataRowData, msg2[0].DataRowData, rows[1].DataRowData)
		}

		br.Proceed()
		msg1, err = readWebsocketRows(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c1 third: %v", err)
		}
		msg2, err = readWebsocketRows(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c2 third: %v", err)
		}
		if len(msg1) != 1 || len(msg2) != 1 {
			t.Fatalf("expected 1 row for each client on third message, got len(msg1)=%d len(msg2)=%d", len(msg1), len(msg2))
		}
		if !reflect.DeepEqual(msg1[0].DataRowData, rows[2].DataRowData) || !reflect.DeepEqual(msg2[0].DataRowData, rows[2].DataRowData) {
			t.Fatalf("third row mismatch: c1=%+v c2=%+v want %+v", msg1[0].DataRowData, msg2[0].DataRowData, rows[2].DataRowData)
		}

		// Close second client; first client should keep receiving.
		closeC2()
		if err := waitWebsocketClosed(c2); err != nil {
			t.Fatalf("wait websocket close c2: %v", err)
		}

		br.Proceed()
		msg1, err = readWebsocketRows(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read websocket c1 fourth: %v", err)
		}
		if len(msg1) != 1 {
			t.Fatalf("expected 1 row for c1 after c2 disconnect, got %d", len(msg1))
		}
		if !reflect.DeepEqual(msg1[0].DataRowData, rows[3].DataRowData) {
			t.Fatalf("c1 row after c2 disconnect mismatch: got %+v want %+v", msg1[0].DataRowData, rows[3].DataRowData)
		}

		if err := waitWebsocketClosed(c1); err != nil {
			t.Fatalf("wait websocket close c1: %v", err)
		}
	})
}

// dialWebSocket2 opens a websocket connection to the /ws2 endpoint for tests.
// Caller is responsible for closing the returned cleanup function.
func dialWebSocket2(baseURL string) (*websocket.Conn, func(), error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse baseURL: %w", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws2"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("dial websocket: %w", err)
	}

	cleanup := func() {
		c.Close(websocket.StatusNormalClosure, "")
	}

	return c, cleanup, nil
}

// readBinaryMessage reads the next binary websocket message with a timeout.
// Returns the raw message bytes and any error encountered.
func readBinaryMessage(c *websocket.Conn, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	typ, data, err := c.Read(ctx)
	if err != nil {
		return nil, err
	}

	if typ != websocket.MessageBinary {
		return nil, fmt.Errorf("expected binary message, got %v", typ)
	}

	return data, nil
}

// waitWebsocketClosed2 waits for a normal websocket closure for /ws2.
func waitWebsocketClosed2(c *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, _, err := c.Read(ctx)
	if err != nil {
		if status := websocket.CloseStatus(err); status != websocket.StatusNormalClosure {
			return fmt.Errorf("unexpected websocket close status: %v", status)
		}
		return nil
	}

	return fmt.Errorf("expected websocket to close, but read succeeded")
}

func TestHTTPServer_WS2_MetadataMessage(t *testing.T) {
	// Test that metadata is sent immediately on connection
	t.Run("SendsMetadataOnConnect", func(t *testing.T) {
		metadata := Metadata{
			WindowSize:    1000,
			XIsTimestamp:  true,
			RelativeStart: false,
			WesplotOptions: WesplotOptions{
				Title:     "Test Chart",
				Columns:   []string{"series1", "series2"},
				XLabel:    "Time",
				YLabel:    "Value",
				YUnit:     "units",
				ChartType: "line",
			},
		}

		ctx := context.Background()
		rows := []DataRow{{DataRowData: DataRowData{X: 1, Ys: []float64{10, 20}}}}
		r := newTestReaderFromRows(rows, 0)
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// First message should be METADATA
		msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read first message: %v", err)
		}

		msg, err := DecodeWSMessage(msgBytes)
		if err != nil {
			t.Fatalf("decode first message: %v", err)
		}

		if msg.Header.Type != MessageTypeMetadata {
			t.Fatalf("expected first message type to be METADATA (0x02), got 0x%02x", msg.Header.Type)
		}

		gotMetadata, ok := msg.Payload.(Metadata)
		if !ok {
			t.Fatalf("expected Metadata payload, got %T", msg.Payload)
		}

		if !reflect.DeepEqual(gotMetadata, metadata) {
			t.Fatalf("metadata mismatch:\nwant: %+v\ngot:  %+v", metadata, gotMetadata)
		}
	})
}

func TestHTTPServer_WS2_DataMessage(t *testing.T) {
	// Test data streaming with single series
	t.Run("SingleSeries", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{20.0}}},
			{DataRowData: DataRowData{X: 3.0, Ys: []float64{30.0}}},
		}
		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// Skip metadata message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}

		// Release all rows
		br.Proceed()
		br.Proceed()
		br.Proceed()

		// Read data message
		msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read data message: %v", err)
		}

		msg, err := DecodeWSMessage(msgBytes)
		if err != nil {
			t.Fatalf("decode data message: %v", err)
		}

		if msg.Header.Type != MessageTypeData {
			t.Fatalf("expected DATA message (0x01), got 0x%02x", msg.Header.Type)
		}

		dataMsg, ok := msg.Payload.(DataMessage)
		if !ok {
			t.Fatalf("expected DataMessage payload, got %T", msg.Payload)
		}

		if dataMsg.SeriesID != 0 {
			t.Fatalf("expected SeriesID 0, got %d", dataMsg.SeriesID)
		}

		if dataMsg.Length != 3 {
			t.Fatalf("expected 3 data points, got %d", dataMsg.Length)
		}

		// Verify X and Y values
		expectedX := []float64{1.0, 2.0, 3.0}
		expectedY := []float64{10.0, 20.0, 30.0}

		if !reflect.DeepEqual(dataMsg.X, expectedX) {
			t.Fatalf("X values mismatch:\nwant: %v\ngot:  %v", expectedX, dataMsg.X)
		}

		if !reflect.DeepEqual(dataMsg.Y, expectedY) {
			t.Fatalf("Y values mismatch:\nwant: %v\ngot:  %v", expectedY, dataMsg.Y)
		}
	})

	// Test data streaming with multiple series
	t.Run("MultipleSeries", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1", "series2", "series3"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0, 100.0, 1000.0}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{20.0, 200.0, 2000.0}}},
		}
		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// Skip metadata message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}

		// Release first row and wait for all series to be sent
		br.Proceed()

		// Read first batch of 3 data messages (historical - one per series for first row)
		firstBatch := make(map[uint32]DataMessage)
		for i := 0; i < 3; i++ {
			msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
			if err != nil {
				t.Fatalf("read first batch message %d: %v", i, err)
			}

			msg, err := DecodeWSMessage(msgBytes)
			if err != nil {
				t.Fatalf("decode first batch message %d: %v", i, err)
			}

			if msg.Header.Type != MessageTypeData {
				t.Fatalf("expected DATA message in first batch, got 0x%02x", msg.Header.Type)
			}

			dataMsg := msg.Payload.(DataMessage)
			firstBatch[dataMsg.SeriesID] = dataMsg
		}

		// Verify first batch has 1 data point per series
		for seriesID := uint32(0); seriesID < 3; seriesID++ {
			dataMsg, ok := firstBatch[seriesID]
			if !ok {
				t.Fatalf("series %d not in first batch", seriesID)
			}
			if dataMsg.Length != 1 {
				t.Fatalf("series %d first batch: expected 1 point, got %d", seriesID, dataMsg.Length)
			}
		}

		// Release second row
		br.Proceed()

		// Read second batch of 3 data messages (one per series for second row)
		secondBatch := make(map[uint32]DataMessage)
		for i := 0; i < 3; i++ {
			msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
			if err != nil {
				t.Fatalf("read second batch message %d: %v", i, err)
			}

			msg, err := DecodeWSMessage(msgBytes)
			if err != nil {
				t.Fatalf("decode second batch message %d: %v", i, err)
			}

			if msg.Header.Type != MessageTypeData {
				t.Fatalf("expected DATA message in second batch, got 0x%02x", msg.Header.Type)
			}

			dataMsg := msg.Payload.(DataMessage)
			secondBatch[dataMsg.SeriesID] = dataMsg
		}

		// Accumulate all data
		receivedSeries := make(map[uint32][]float64)
		for seriesID := uint32(0); seriesID < 3; seriesID++ {
			var allX, allY []float64
			if firstMsg, ok := firstBatch[seriesID]; ok {
				allX = append(allX, firstMsg.X...)
				allY = append(allY, firstMsg.Y...)
			}
			if secondMsg, ok := secondBatch[seriesID]; ok {
				allX = append(allX, secondMsg.X...)
				allY = append(allY, secondMsg.Y...)
			}
			receivedSeries[seriesID] = allY
		}

		// Verify series 0
		expectedY0 := []float64{10.0, 20.0}
		if !reflect.DeepEqual(receivedSeries[0], expectedY0) {
			t.Fatalf("series 0 mismatch: got Y=%v, want %v", receivedSeries[0], expectedY0)
		}

		// Verify series 1
		expectedY1 := []float64{100.0, 200.0}
		if !reflect.DeepEqual(receivedSeries[1], expectedY1) {
			t.Fatalf("series 1 mismatch: got Y=%v, want %v", receivedSeries[1], expectedY1)
		}

		// Verify series 2
		expectedY2 := []float64{1000.0, 2000.0}
		if !reflect.DeepEqual(receivedSeries[2], expectedY2) {
			t.Fatalf("series 2 mismatch: got Y=%v, want %v", receivedSeries[2], expectedY2)
		}
	})

	// Test empty data (edge case)
	t.Run("EmptyData", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{} // No data rows
		r := newTestReaderFromRows(rows, 0)
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// Skip metadata message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}

		// Should receive STREAM_END immediately
		msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read stream end: %v", err)
		}

		msg, err := DecodeWSMessage(msgBytes)
		if err != nil {
			t.Fatalf("decode stream end: %v", err)
		}

		if msg.Header.Type != MessageTypeStreamEnd {
			t.Fatalf("expected STREAM_END message (0x03), got 0x%02x", msg.Header.Type)
		}

		streamEnd, ok := msg.Payload.(StreamEndMessage)
		if !ok {
			t.Fatalf("expected StreamEndMessage payload, got %T", msg.Payload)
		}

		if streamEnd.Error {
			t.Fatalf("expected clean stream end, got error: %s", streamEnd.Msg)
		}
	})
}

func TestHTTPServer_WS2_StreamEnd(t *testing.T) {
	// Test clean stream end
	t.Run("CleanEnd", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0}}},
		}
		r := newTestReaderFromRows(rows, 0)
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// Skip metadata message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}

		// Skip data message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read data: %v", err)
		}

		// Read stream end message
		msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read stream end: %v", err)
		}

		msg, err := DecodeWSMessage(msgBytes)
		if err != nil {
			t.Fatalf("decode stream end: %v", err)
		}

		if msg.Header.Type != MessageTypeStreamEnd {
			t.Fatalf("expected STREAM_END message (0x03), got 0x%02x", msg.Header.Type)
		}

		streamEnd, ok := msg.Payload.(StreamEndMessage)
		if !ok {
			t.Fatalf("expected StreamEndMessage payload, got %T", msg.Payload)
		}

		if streamEnd.Error {
			t.Fatalf("expected clean stream end, got error: %s", streamEnd.Msg)
		}

		// Verify websocket closes
		if err := waitWebsocketClosed2(c); err != nil {
			t.Fatalf("wait websocket close: %v", err)
		}
	})

	// Test stream end with error
	t.Run("WithError", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0}}},
		}
		r := newTestReaderFromItems([]interface{}{
			rows[0],
			fmt.Errorf("test error"),
		})
		d := NewDataBroadcaster(r, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		c, closeConn, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket: %v", err)
		}
		defer closeConn()

		// Skip metadata message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read metadata: %v", err)
		}

		// Skip data message
		_, err = readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read data: %v", err)
		}

		// Read stream end message
		msgBytes, err := readBinaryMessage(c, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read stream end: %v", err)
		}

		msg, err := DecodeWSMessage(msgBytes)
		if err != nil {
			t.Fatalf("decode stream end: %v", err)
		}

		if msg.Header.Type != MessageTypeStreamEnd {
			t.Fatalf("expected STREAM_END message (0x03), got 0x%02x", msg.Header.Type)
		}

		streamEnd, ok := msg.Payload.(StreamEndMessage)
		if !ok {
			t.Fatalf("expected StreamEndMessage payload, got %T", msg.Payload)
		}

		if !streamEnd.Error {
			t.Fatal("expected error stream end, got clean end")
		}

		if streamEnd.Msg != "test error" {
			t.Fatalf("expected error message 'test error', got '%s'", streamEnd.Msg)
		}

		// Verify websocket closes
		if err := waitWebsocketClosed2(c); err != nil {
			t.Fatalf("wait websocket close: %v", err)
		}
	})
}

func TestHTTPServer_WS2_MultipleClients(t *testing.T) {
	// Test that multiple clients can connect and receive data independently
	t.Run("IndependentClients", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Columns: []string{"series1"},
			},
		}

		ctx := context.Background()
		rows := []DataRow{
			{DataRowData: DataRowData{X: 1.0, Ys: []float64{10.0}}},
			{DataRowData: DataRowData{X: 2.0, Ys: []float64{20.0}}},
		}
		br := &blockingDataRowReader{rows: rows, proceed: make(chan struct{})}
		d := NewDataBroadcaster(br, 10, nil)
		d.Start(ctx)

		baseURL, cleanup := startTestServer(metadata, d)
		defer cleanup()

		// Connect first client
		c1, closeC1, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket c1: %v", err)
		}
		defer closeC1()

		// Skip metadata for c1
		_, err = readBinaryMessage(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c1 metadata: %v", err)
		}

		// Let first row through
		br.Proceed()

		// Read first data message on c1
		msgBytes1, err := readBinaryMessage(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c1 first data: %v", err)
		}

		msg1, err := DecodeWSMessage(msgBytes1)
		if err != nil {
			t.Fatalf("decode c1 first data: %v", err)
		}

		dataMsg1, ok := msg1.Payload.(DataMessage)
		if !ok {
			t.Fatalf("expected DataMessage, got %T", msg1.Payload)
		}

		if dataMsg1.Length != 1 || dataMsg1.X[0] != 1.0 || dataMsg1.Y[0] != 10.0 {
			t.Fatalf("c1 first data mismatch: got X=%v Y=%v", dataMsg1.X, dataMsg1.Y)
		}

		// Connect second client
		c2, closeC2, err := dialWebSocket2(baseURL)
		if err != nil {
			t.Fatalf("dial websocket c2: %v", err)
		}
		defer closeC2()

		// c2 should receive metadata
		_, err = readBinaryMessage(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c2 metadata: %v", err)
		}

		// c2 should receive historical data (first row)
		msgBytes2, err := readBinaryMessage(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c2 historical data: %v", err)
		}

		msg2, err := DecodeWSMessage(msgBytes2)
		if err != nil {
			t.Fatalf("decode c2 historical data: %v", err)
		}

		dataMsg2, ok := msg2.Payload.(DataMessage)
		if !ok {
			t.Fatalf("expected DataMessage, got %T", msg2.Payload)
		}

		if dataMsg2.Length != 1 || dataMsg2.X[0] != 1.0 || dataMsg2.Y[0] != 10.0 {
			t.Fatalf("c2 historical data mismatch: got X=%v Y=%v", dataMsg2.X, dataMsg2.Y)
		}

		// Let second row through - both clients should receive it
		br.Proceed()

		msgBytes1, err = readBinaryMessage(c1, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c1 second data: %v", err)
		}

		msgBytes2, err = readBinaryMessage(c2, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("read c2 second data: %v", err)
		}

		msg1, _ = DecodeWSMessage(msgBytes1)
		msg2, _ = DecodeWSMessage(msgBytes2)

		dataMsg1 = msg1.Payload.(DataMessage)
		dataMsg2 = msg2.Payload.(DataMessage)

		if dataMsg1.Length != 1 || dataMsg1.X[0] != 2.0 || dataMsg1.Y[0] != 20.0 {
			t.Fatalf("c1 second data mismatch: got X=%v Y=%v", dataMsg1.X, dataMsg1.Y)
		}

		if dataMsg2.Length != 1 || dataMsg2.X[0] != 2.0 || dataMsg2.Y[0] != 20.0 {
			t.Fatalf("c2 second data mismatch: got X=%v Y=%v", dataMsg2.X, dataMsg2.Y)
		}
	})
}
