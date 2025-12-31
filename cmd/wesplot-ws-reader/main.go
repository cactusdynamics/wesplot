package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strconv"

	"github.com/cactusdynamics/wesplot"
	"nhooyr.io/websocket"
)

// Config holds the configuration for the WS reader
type Config struct {
	ServerURL string
	Output    io.Writer
	Logger    *slog.Logger
}

// WSReader reads from wesplot WS2 endpoint and outputs CSV data
type WSReader struct {
	config    Config
	csvWriter *csv.Writer
}

// NewWSReader creates a new WS reader with the given configuration
func NewWSReader(config Config) *WSReader {
	return &WSReader{
		config:    config,
		csvWriter: csv.NewWriter(config.Output),
	}
}

// Connect establishes websocket connection and processes messages
func (w *WSReader) Connect() error {
	u, err := url.Parse(w.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// Change scheme to websocket
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	// Add /ws2 endpoint
	u.Path = "/ws2"

	w.config.Logger.Info("Connecting to websocket", "url", u.String())

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Write CSV header
	if err := w.csvWriter.Write([]string{"series_id", "x", "y"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for {
		_, messageData, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				w.config.Logger.Info("Connection closed normally")
				break
			}
			w.config.Logger.Error("Error reading message", "error", err)
			break
		}

		if err := w.processMessage(messageData); err != nil {
			if err == io.EOF {
				w.config.Logger.Info("Stream ended")
				break
			}
			w.config.Logger.Error("Error processing message", "error", err)
		}
	}

	w.csvWriter.Flush()
	return w.csvWriter.Error()
}

// processMessage processes a single websocket message
func (w *WSReader) processMessage(messageData []byte) error {
	msg, err := wesplot.DecodeWSMessage(messageData)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	switch msg.Header.Type {
	case wesplot.MessageTypeData:
		dataMsg, ok := msg.Payload.(wesplot.DataMessage)
		if !ok {
			return fmt.Errorf("invalid DATA message payload type: %T", msg.Payload)
		}
		return w.processDataMessage(dataMsg)

	case wesplot.MessageTypeMetadata:
		metadata, ok := msg.Payload.(wesplot.Metadata)
		if !ok {
			return fmt.Errorf("invalid METADATA message payload type: %T", msg.Payload)
		}
		w.config.Logger.Debug("Received metadata", "metadata", metadata)

	case wesplot.MessageTypeStreamEnd:
		streamEnd, ok := msg.Payload.(wesplot.StreamEndMessage)
		if !ok {
			return fmt.Errorf("invalid STREAM_END message payload type: %T", msg.Payload)
		}
		if streamEnd.Error {
			w.config.Logger.Error("Stream ended with error", "message", streamEnd.Msg)
		} else {
			w.config.Logger.Info("Stream ended successfully", "message", streamEnd.Msg)
		}
		return io.EOF // Signal end of stream

	default:
		w.config.Logger.Warn("Unknown message type", "type", fmt.Sprintf("0x%02x", msg.Header.Type))
	}

	return nil
}

// processDataMessage processes a DATA message and writes CSV rows
func (w *WSReader) processDataMessage(dataMsg wesplot.DataMessage) error {
	seriesID := strconv.FormatUint(uint64(dataMsg.SeriesID), 10)

	for i := 0; i < len(dataMsg.X); i++ {
		row := []string{
			seriesID,
			strconv.FormatFloat(dataMsg.X[i], 'g', -1, 64),
			strconv.FormatFloat(dataMsg.Y[i], 'g', -1, 64),
		}
		if err := w.csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	w.csvWriter.Flush()
	return w.csvWriter.Error()
}

func main() {
	var serverURL = flag.String("url", "http://localhost:5274", "URL of the wesplot server")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	config := Config{
		ServerURL: *serverURL,
		Output:    os.Stdout,
		Logger:    logger,
	}

	reader := NewWSReader(config)
	if err := reader.Connect(); err != nil {
		config.Logger.Error("Failed to connect", "error", err)
		os.Exit(1)
	}
}
