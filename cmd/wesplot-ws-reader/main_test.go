package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cactusdynamics/wesplot"
)

// TestWSReaderBasicData tests basic data reading functionality
func TestWSReaderBasicData(t *testing.T) {
	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Create test metadata
	metadata := wesplot.Metadata{
		WindowSize:    100,
		XIsTimestamp:  false,
		RelativeStart: false,
		WesplotOptions: wesplot.WesplotOptions{
			Title:   "Test Data",
			Columns: []string{"Series1", "Series2"},
		},
	}

	// Create mock data broadcaster with test data
	testDataRows := []wesplot.DataRow{
		{DataRowData: wesplot.DataRowData{X: 1.0, Ys: []float64{10.5, 20.3}}},
		{DataRowData: wesplot.DataRowData{X: 2.0, Ys: []float64{11.2, 21.1}}},
		{DataRowData: wesplot.DataRowData{X: 3.0, Ys: []float64{12.8, 19.7}}},
	}

	mockReader := &MockDataRowReader{
		data:    testDataRows,
		columns: []string{"Series1", "Series2"},
	}
	dataBroadcaster := wesplot.NewDataBroadcaster(mockReader, metadata.WindowSize, nil)

	// Create HTTP server
	server := wesplot.NewHttpServer(dataBroadcaster, "localhost", uint16(port), metadata, 50*time.Millisecond)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataBroadcaster.Start(ctx)

	go func() {
		server.Run()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create output buffer
	var output bytes.Buffer
	errorBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(errorBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := Config{
		ServerURL: "http://localhost:" + strconv.Itoa(port),
		Output:    &output,
		Logger:    logger,
	}

	reader := NewWSReader(config)

	// Connect and read data (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- reader.Connect()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WSReader.Connect() failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WSReader.Connect() timed out")
	}

	// Parse CSV output
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")

	// Check CSV header
	if len(lines) < 1 {
		t.Fatal("No CSV output received")
	}

	expectedHeader := "series_id,x,y"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header %q, got %q", expectedHeader, lines[0])
	}

	// Check data rows
	expectedRows := []string{
		"0,1,10.5",
		"1,1,20.3",
		"0,2,11.2",
		"1,2,21.1",
		"0,3,12.8",
		"1,3,19.7",
	}

	dataLines := lines[1:]
	if len(dataLines) < len(expectedRows) {
		t.Errorf("Expected at least %d data rows, got %d", len(expectedRows), len(dataLines))
	}

	// Check that expected rows are present (order might vary due to concurrency)
	for _, expectedRow := range expectedRows {
		found := false
		for _, dataLine := range dataLines {
			if dataLine == expectedRow {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected row %q not found in output", expectedRow)
		}
	}
}

// TestWSReaderEmptyData tests handling of empty data messages
func TestWSReaderEmptyData(t *testing.T) {
	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Create test metadata
	metadata := wesplot.Metadata{
		WindowSize:    100,
		XIsTimestamp:  false,
		RelativeStart: false,
		WesplotOptions: wesplot.WesplotOptions{
			Title:   "Test Empty Data",
			Columns: []string{"Series1"},
		},
	}

	// Create mock data broadcaster with empty data
	mockReader := &MockDataRowReader{
		data:    []wesplot.DataRow{},
		columns: []string{"Series1"},
	}
	dataBroadcaster := wesplot.NewDataBroadcaster(mockReader, metadata.WindowSize, nil)

	// Create HTTP server
	server := wesplot.NewHttpServer(dataBroadcaster, "localhost", uint16(port), metadata, 50*time.Millisecond)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataBroadcaster.Start(ctx)

	go func() {
		server.Run()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create output buffer
	var output bytes.Buffer
	errorBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(errorBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := Config{
		ServerURL: "http://localhost:" + strconv.Itoa(port),
		Output:    &output,
		Logger:    logger,
	}

	reader := NewWSReader(config)

	// Connect and read data (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- reader.Connect()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WSReader.Connect() failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WSReader.Connect() timed out")
	}

	// Parse CSV output - should only have header
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")

	if len(lines) != 1 {
		t.Errorf("Expected only header line, got %d lines", len(lines))
	}

	expectedHeader := "series_id,x,y"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header %q, got %q", expectedHeader, lines[0])
	}
}

// MockDataRowReader is a test implementation of DataRowReader
type MockDataRowReader struct {
	data    []wesplot.DataRow
	columns []string
	index   int
}

func (m *MockDataRowReader) Read(ctx context.Context) (wesplot.DataRow, error) {
	if m.index >= len(m.data) {
		// Send EOF to signal end of data
		return wesplot.DataRow{}, io.EOF
	}

	row := m.data[m.index]
	m.index++
	return row, nil
}

func (m *MockDataRowReader) ColumnNames() []string {
	return m.columns
}

func (m *MockDataRowReader) Reset() {
	m.index = 0
}
