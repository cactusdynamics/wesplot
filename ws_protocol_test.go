package wesplot

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestEncodeDecodeEnvelopeHeader tests envelope header encoding and decoding round-trip
func TestEncodeDecodeEnvelopeHeader(t *testing.T) {
	tests := []struct {
		name string
		env  EnvelopeHeader
	}{
		{
			name: "basic envelope",
			env: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeData,
				Length:  1024,
			},
		},
		{
			name: "zero length payload",
			env: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeMetadata,
				Length:  0,
			},
		},
		{
			name: "large payload",
			env: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeStreamEnd,
				Length:  1000000,
			},
		},
		{
			name: "envelope with reserved bytes",
			env: EnvelopeHeader{
				Version:  ProtocolVersion,
				Reserved: [2]byte{0xAB, 0xCD},
				Type:     MessageTypeData,
				Length:   512,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := EncodeEnvelopeHeader(tt.env)

			// Verify header size
			if len(encoded) != EnvelopeHeaderSize {
				t.Errorf("encoded header size = %d, want %d", len(encoded), EnvelopeHeaderSize)
			}

			// Decode
			decoded, err := DecodeEnvelopeHeader(encoded)
			if err != nil {
				t.Fatalf("DecodeEnvelopeHeader() error = %v", err)
			}

			// Verify fields
			if decoded.Version != tt.env.Version {
				t.Errorf("Version = %d, want %d", decoded.Version, tt.env.Version)
			}
			if decoded.Reserved != tt.env.Reserved {
				t.Errorf("Reserved = %v, want %v", decoded.Reserved, tt.env.Reserved)
			}
			if decoded.Type != tt.env.Type {
				t.Errorf("Type = %d, want %d", decoded.Type, tt.env.Type)
			}
			if decoded.Length != tt.env.Length {
				t.Errorf("Length = %d, want %d", decoded.Length, tt.env.Length)
			}
		})
	}
}

// TestDecodeEnvelopeHeaderErrors tests error cases for envelope header decoding
func TestDecodeEnvelopeHeaderErrors(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		errContains string
	}{
		{
			name:        "buffer too short - empty",
			buf:         []byte{},
			errContains: "buffer too short",
		},
		{
			name:        "buffer too short - 7 bytes",
			buf:         []byte{1, 2, 3, 4, 5, 6, 7},
			errContains: "buffer too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeEnvelopeHeader(tt.buf)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestEncodeDecodeDataMessage tests DATA message encoding and decoding
func TestEncodeDecodeDataMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  DataMessage
	}{
		{
			name: "single point",
			msg: DataMessage{
				SeriesID: 0,
				Length:   1,
				X:        []float64{1.0},
				Y:        []float64{10.5},
			},
		},
		{
			name: "multiple points",
			msg: DataMessage{
				SeriesID: 1,
				Length:   3,
				X:        []float64{1.0, 2.0, 3.0},
				Y:        []float64{10.5, 20.3, 15.7},
			},
		},
		{
			name: "empty arrays (series break)",
			msg: DataMessage{
				SeriesID: 2,
				Length:   0,
				X:        []float64{},
				Y:        []float64{},
			},
		},
		{
			name: "large dataset",
			msg: DataMessage{
				SeriesID: 0,
				Length:   1000,
				X:        makeSampleData(1000),
				Y:        makeSampleData(1000),
			},
		},
		{
			name: "special float values",
			msg: DataMessage{
				SeriesID: 0,
				Length:   5,
				X:        []float64{0.0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1), math.NaN()},
				Y:        []float64{math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64, 1e-308, 1e308},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeDataMessage(tt.msg)
			if err != nil {
				t.Fatalf("EncodeDataMessage() error = %v", err)
			}

			// Verify payload size
			expectedSize := 8 + (tt.msg.Length * 8 * 2)
			if uint32(len(encoded)) != expectedSize {
				t.Errorf("encoded size = %d, want %d", len(encoded), expectedSize)
			}

			// Decode
			decoded, err := DecodeDataMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeDataMessage() error = %v", err)
			}

			// Verify fields
			if decoded.SeriesID != tt.msg.SeriesID {
				t.Errorf("SeriesID = %d, want %d", decoded.SeriesID, tt.msg.SeriesID)
			}
			if decoded.Length != tt.msg.Length {
				t.Errorf("Length = %d, want %d", decoded.Length, tt.msg.Length)
			}

			// Verify X array
			if len(decoded.X) != len(tt.msg.X) {
				t.Errorf("X length = %d, want %d", len(decoded.X), len(tt.msg.X))
			}
			for i := range tt.msg.X {
				if !floatEqual(decoded.X[i], tt.msg.X[i]) {
					t.Errorf("X[%d] = %v, want %v", i, decoded.X[i], tt.msg.X[i])
				}
			}

			// Verify Y array
			if len(decoded.Y) != len(tt.msg.Y) {
				t.Errorf("Y length = %d, want %d", len(decoded.Y), len(tt.msg.Y))
			}
			for i := range tt.msg.Y {
				if !floatEqual(decoded.Y[i], tt.msg.Y[i]) {
					t.Errorf("Y[%d] = %v, want %v", i, decoded.Y[i], tt.msg.Y[i])
				}
			}
		})
	}
}

// TestEncodeDataMessageErrors tests error cases for DATA message encoding
func TestEncodeDataMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		msg         DataMessage
		errContains string
	}{
		{
			name: "X and Y length mismatch",
			msg: DataMessage{
				SeriesID: 0,
				Length:   2,
				X:        []float64{1.0, 2.0},
				Y:        []float64{10.0},
			},
			errContains: "must have same length",
		},
		{
			name: "Length field doesn't match array length",
			msg: DataMessage{
				SeriesID: 0,
				Length:   5,
				X:        []float64{1.0, 2.0},
				Y:        []float64{10.0, 20.0},
			},
			errContains: "doesn't match array length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeDataMessage(tt.msg)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestDecodeDataMessageErrors tests error cases for DATA message decoding
func TestDecodeDataMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		errContains string
	}{
		{
			name:        "buffer too short - empty",
			buf:         []byte{},
			errContains: "buffer too short",
		},
		{
			name:        "buffer too short - 7 bytes",
			buf:         []byte{1, 2, 3, 4, 5, 6, 7},
			errContains: "buffer too short",
		},
		{
			name: "buffer size mismatch - missing data",
			buf: func() []byte {
				// Encode a valid header but provide incomplete data
				buf := make([]byte, 8)
				// SeriesID = 0
				// Length = 10 (but we don't provide 10 pairs)
				buf[4] = 10
				return buf
			}(),
			errContains: "buffer size mismatch",
		},
		{
			name: "buffer size mismatch - too much data",
			buf: func() []byte {
				// Encode a header with Length=1 but provide data for 3 pairs
				buf := make([]byte, 8+3*8*2)
				// SeriesID = 0
				// Length = 1 (but buffer has space for 3 pairs)
				buf[4] = 1
				return buf
			}(),
			errContains: "buffer size mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeDataMessage(tt.buf)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestEncodeDecodeMetadataMessage tests METADATA message encoding and decoding
func TestEncodeDecodeMetadataMessage(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
	}{
		{
			name: "basic metadata",
			metadata: Metadata{
				WindowSize:    1000,
				XIsTimestamp:  true,
				RelativeStart: false,
				WesplotOptions: WesplotOptions{
					Title:     "System Metrics",
					Columns:   []string{"CPU", "Memory"},
					XLabel:    "Time",
					YLabel:    "Usage",
					YUnit:     "%",
					ChartType: "line",
				},
			},
		},
		{
			name: "minimal metadata",
			metadata: Metadata{
				WindowSize: 0,
				WesplotOptions: WesplotOptions{
					Columns: []string{"Data"},
				},
			},
		},
		{
			name: "metadata with optional fields",
			metadata: Metadata{
				WindowSize:    500,
				XIsTimestamp:  false,
				RelativeStart: true,
				WesplotOptions: WesplotOptions{
					Title:     "Test Chart",
					Columns:   []string{"A", "B", "C"},
					XLabel:    "X",
					YLabel:    "Y",
					YMin:      floatPtr(0.0),
					YMax:      floatPtr(100.0),
					YUnit:     "units",
					ChartType: "bar",
				},
			},
		},
		{
			name: "many columns",
			metadata: Metadata{
				WindowSize: 100,
				WesplotOptions: WesplotOptions{
					Columns: []string{"C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9", "C10"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeMetadataMessage(tt.metadata)
			if err != nil {
				t.Fatalf("EncodeMetadataMessage() error = %v", err)
			}

			// Decode
			decoded, err := DecodeMetadataMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeMetadataMessage() error = %v", err)
			}

			// Verify using deep equal
			if !reflect.DeepEqual(decoded, tt.metadata) {
				t.Errorf("Decoded metadata does not match original.\nGot:  %+v\nWant: %+v", decoded, tt.metadata)
			}
		})
	}
}

// TestDecodeMetadataMessageErrors tests error cases for METADATA message decoding
func TestDecodeMetadataMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		errContains string
	}{
		{
			name:        "buffer too short - empty",
			buf:         []byte{},
			errContains: "buffer too short",
		},
		{
			name:        "buffer too short - 3 bytes",
			buf:         []byte{1, 2, 3},
			errContains: "buffer too short",
		},
		{
			name: "buffer size mismatch",
			buf: func() []byte {
				// Length says 100 bytes but we provide less
				buf := make([]byte, 10)
				buf[0] = 100 // JSON length
				return buf
			}(),
			errContains: "buffer size mismatch",
		},
		{
			name: "invalid JSON",
			buf: func() []byte {
				invalidJSON := []byte("{invalid json")
				buf := make([]byte, 4+len(invalidJSON))
				buf[0] = byte(len(invalidJSON))
				copy(buf[4:], invalidJSON)
				return buf
			}(),
			errContains: "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeMetadataMessage(tt.buf)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestEncodeDecodeStreamEndMessage tests STREAM_END message encoding and decoding
func TestEncodeDecodeStreamEndMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  StreamEndMessage
	}{
		{
			name: "clean end - no message",
			msg: StreamEndMessage{
				Error: false,
				Msg:   "",
			},
		},
		{
			name: "clean end - with message",
			msg: StreamEndMessage{
				Error: false,
				Msg:   "Stream completed successfully",
			},
		},
		{
			name: "error end",
			msg: StreamEndMessage{
				Error: true,
				Msg:   "Failed to read from stdin: EOF",
			},
		},
		{
			name: "error with long message",
			msg: StreamEndMessage{
				Error: true,
				Msg:   "A very long error message that describes in detail what went wrong with the stream processing and provides context for debugging purposes.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeStreamEndMessage(tt.msg)
			if err != nil {
				t.Fatalf("EncodeStreamEndMessage() error = %v", err)
			}

			// Decode
			decoded, err := DecodeStreamEndMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeStreamEndMessage() error = %v", err)
			}

			// Verify using deep equal
			if !reflect.DeepEqual(decoded, tt.msg) {
				t.Errorf("Decoded message does not match original.\nGot:  %+v\nWant: %+v", decoded, tt.msg)
			}
		})
	}
}

// TestDecodeStreamEndMessageErrors tests error cases for STREAM_END message decoding
func TestDecodeStreamEndMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		errContains string
	}{
		{
			name:        "buffer too short - empty",
			buf:         []byte{},
			errContains: "buffer too short",
		},
		{
			name:        "buffer too short - 3 bytes",
			buf:         []byte{1, 2, 3},
			errContains: "buffer too short",
		},
		{
			name: "buffer size mismatch",
			buf: func() []byte {
				// Length says 50 bytes but we provide less
				buf := make([]byte, 10)
				buf[0] = 50 // JSON length
				return buf
			}(),
			errContains: "buffer size mismatch",
		},
		{
			name: "invalid JSON",
			buf: func() []byte {
				invalidJSON := []byte("{not valid json")
				buf := make([]byte, 4+len(invalidJSON))
				buf[0] = byte(len(invalidJSON))
				copy(buf[4:], invalidJSON)
				return buf
			}(),
			errContains: "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeStreamEndMessage(tt.buf)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestEncodeDecodeWSMessage tests WSMessage encoding and decoding
func TestEncodeDecodeWSMessage(t *testing.T) {
	// Test DATA message
	t.Run("DATA message", func(t *testing.T) {
		dataMsg := DataMessage{
			SeriesID: 0,
			Length:   2,
			X:        []float64{1.0, 2.0},
			Y:        []float64{10.0, 20.0},
		}

		wsMsg := WSMessage{
			Header: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeData,
			},
			Payload: dataMsg,
		}

		// Encode
		fullMsg, err := EncodeWSMessage(wsMsg)
		if err != nil {
			t.Fatalf("EncodeWSMessage() error = %v", err)
		}

		// Decode
		decoded, err := DecodeWSMessage(fullMsg)
		if err != nil {
			t.Fatalf("DecodeWSMessage() error = %v", err)
		}

		// Verify header
		if decoded.Header.Version != ProtocolVersion {
			t.Errorf("Version = %d, want %d", decoded.Header.Version, ProtocolVersion)
		}
		if decoded.Header.Type != MessageTypeData {
			t.Errorf("Type = %d, want %d", decoded.Header.Type, MessageTypeData)
		}

		// Verify payload
		decodedData, ok := decoded.Payload.(DataMessage)
		if !ok {
			t.Fatalf("Payload type = %T, want DataMessage", decoded.Payload)
		}

		if decodedData.SeriesID != dataMsg.SeriesID {
			t.Errorf("SeriesID = %d, want %d", decodedData.SeriesID, dataMsg.SeriesID)
		}
		if decodedData.Length != dataMsg.Length {
			t.Errorf("Length = %d, want %d", decodedData.Length, dataMsg.Length)
		}
		if !reflect.DeepEqual(decodedData.X, dataMsg.X) {
			t.Errorf("X = %v, want %v", decodedData.X, dataMsg.X)
		}
		if !reflect.DeepEqual(decodedData.Y, dataMsg.Y) {
			t.Errorf("Y = %v, want %v", decodedData.Y, dataMsg.Y)
		}
	})

	// Test METADATA message
	t.Run("METADATA message", func(t *testing.T) {
		metadata := Metadata{
			WindowSize: 1000,
			WesplotOptions: WesplotOptions{
				Title:   "Test",
				Columns: []string{"A", "B"},
			},
		}

		wsMsg := WSMessage{
			Header: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeMetadata,
			},
			Payload: metadata,
		}

		// Encode
		fullMsg, err := EncodeWSMessage(wsMsg)
		if err != nil {
			t.Fatalf("EncodeWSMessage() error = %v", err)
		}

		// Decode
		decoded, err := DecodeWSMessage(fullMsg)
		if err != nil {
			t.Fatalf("DecodeWSMessage() error = %v", err)
		}

		if decoded.Header.Type != MessageTypeMetadata {
			t.Errorf("Type = %d, want %d", decoded.Header.Type, MessageTypeMetadata)
		}

		// Verify payload
		decodedMetadata, ok := decoded.Payload.(Metadata)
		if !ok {
			t.Fatalf("Payload type = %T, want Metadata", decoded.Payload)
		}

		if !reflect.DeepEqual(decodedMetadata, metadata) {
			t.Errorf("Decoded metadata does not match original.\nGot:  %+v\nWant: %+v", decodedMetadata, metadata)
		}
	})

	// Test STREAM_END message
	t.Run("STREAM_END message", func(t *testing.T) {
		streamEnd := StreamEndMessage{
			Error: true,
			Msg:   "Error occurred",
		}

		wsMsg := WSMessage{
			Header: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeStreamEnd,
			},
			Payload: streamEnd,
		}

		// Encode
		fullMsg, err := EncodeWSMessage(wsMsg)
		if err != nil {
			t.Fatalf("EncodeWSMessage() error = %v", err)
		}

		// Decode
		decoded, err := DecodeWSMessage(fullMsg)
		if err != nil {
			t.Fatalf("DecodeWSMessage() error = %v", err)
		}

		if decoded.Header.Type != MessageTypeStreamEnd {
			t.Errorf("Type = %d, want %d", decoded.Header.Type, MessageTypeStreamEnd)
		}

		// Verify payload
		decodedStreamEnd, ok := decoded.Payload.(StreamEndMessage)
		if !ok {
			t.Fatalf("Payload type = %T, want StreamEndMessage", decoded.Payload)
		}

		if decodedStreamEnd.Error != streamEnd.Error {
			t.Errorf("Error = %v, want %v", decodedStreamEnd.Error, streamEnd.Error)
		}
		if decodedStreamEnd.Msg != streamEnd.Msg {
			t.Errorf("Msg = %s, want %s", decodedStreamEnd.Msg, streamEnd.Msg)
		}
	})

	// Test empty DATA message (series break)
	t.Run("empty DATA message", func(t *testing.T) {
		dataMsg := DataMessage{
			SeriesID: 5,
			Length:   0,
			X:        []float64{},
			Y:        []float64{},
		}

		wsMsg := WSMessage{
			Header: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeData,
			},
			Payload: dataMsg,
		}

		// Encode
		fullMsg, err := EncodeWSMessage(wsMsg)
		if err != nil {
			t.Fatalf("EncodeWSMessage() error = %v", err)
		}

		// Decode
		decoded, err := DecodeWSMessage(fullMsg)
		if err != nil {
			t.Fatalf("DecodeWSMessage() error = %v", err)
		}

		decodedData, ok := decoded.Payload.(DataMessage)
		if !ok {
			t.Fatalf("Payload type = %T, want DataMessage", decoded.Payload)
		}

		if decodedData.SeriesID != dataMsg.SeriesID {
			t.Errorf("SeriesID = %d, want %d", decodedData.SeriesID, dataMsg.SeriesID)
		}
		if decodedData.Length != 0 {
			t.Errorf("Length = %d, want 0", decodedData.Length)
		}
	})

	// Test with reserved bytes
	t.Run("message with reserved bytes", func(t *testing.T) {
		dataMsg := DataMessage{
			SeriesID: 1,
			Length:   1,
			X:        []float64{5.0},
			Y:        []float64{10.0},
		}

		wsMsg := WSMessage{
			Header: EnvelopeHeader{
				Version:  ProtocolVersion,
				Reserved: [2]byte{0xAB, 0xCD},
				Type:     MessageTypeData,
			},
			Payload: dataMsg,
		}

		// Encode
		fullMsg, err := EncodeWSMessage(wsMsg)
		if err != nil {
			t.Fatalf("EncodeWSMessage() error = %v", err)
		}

		// Decode
		decoded, err := DecodeWSMessage(fullMsg)
		if err != nil {
			t.Fatalf("DecodeWSMessage() error = %v", err)
		}

		// Verify reserved bytes are preserved
		if decoded.Header.Reserved != wsMsg.Header.Reserved {
			t.Errorf("Reserved = %v, want %v", decoded.Header.Reserved, wsMsg.Header.Reserved)
		}
	})
}

// TestDecodeWSMessageErrors tests error cases for WSMessage decoding
func TestDecodeWSMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		buf         []byte
		errContains string
	}{
		{
			name:        "buffer too short for header",
			buf:         []byte{1, 2, 3},
			errContains: "buffer too short",
		},
		{
			name: "buffer too short for payload",
			buf: func() []byte {
				env := EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeData,
					Length:  1000, // Claims 1000 bytes but we don't provide them
				}
				return EncodeEnvelopeHeader(env)
			}(),
			errContains: "buffer too short",
		},
		{
			name: "unknown message type",
			buf: func() []byte {
				env := EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    0xFF, // Invalid type
					Length:  0,
				}
				return EncodeEnvelopeHeader(env)
			}(),
			errContains: "unknown message type",
		},
		{
			name: "invalid DATA payload",
			buf: func() []byte {
				env := EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeData,
					Length:  4, // Too short for valid DATA message
				}
				header := EncodeEnvelopeHeader(env)
				fullMsg := make([]byte, len(header)+4)
				copy(fullMsg, header)
				return fullMsg
			}(),
			errContains: "buffer too short for DATA message",
		},
		{
			name: "invalid METADATA payload",
			buf: func() []byte {
				env := EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeMetadata,
					Length:  4, // Too short for valid METADATA message
				}
				header := EncodeEnvelopeHeader(env)
				fullMsg := make([]byte, len(header)+4)
				copy(fullMsg, header)
				return fullMsg
			}(),
			errContains: "failed to unmarshal metadata",
		},
		{
			name: "invalid STREAM_END payload",
			buf: func() []byte {
				env := EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeStreamEnd,
					Length:  4, // Too short for valid STREAM_END message
				}
				header := EncodeEnvelopeHeader(env)
				fullMsg := make([]byte, len(header)+4)
				copy(fullMsg, header)
				return fullMsg
			}(),
			errContains: "failed to unmarshal stream end message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeWSMessage(tt.buf)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestEncodeWSMessageErrors tests error cases for WSMessage encoding
func TestEncodeWSMessageErrors(t *testing.T) {
	tests := []struct {
		name        string
		msg         WSMessage
		errContains string
	}{
		{
			name: "payload type mismatch - wrong type for DATA",
			msg: WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeData,
				},
				Payload: Metadata{}, // Wrong payload type
			},
			errContains: "payload type mismatch",
		},
		{
			name: "payload type mismatch - wrong type for METADATA",
			msg: WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeMetadata,
				},
				Payload: DataMessage{}, // Wrong payload type
			},
			errContains: "payload type mismatch",
		},
		{
			name: "payload type mismatch - wrong type for STREAM_END",
			msg: WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeStreamEnd,
				},
				Payload: DataMessage{}, // Wrong payload type
			},
			errContains: "payload type mismatch",
		},
		{
			name: "unknown message type",
			msg: WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    0xFF, // Invalid type
				},
				Payload: DataMessage{},
			},
			errContains: "unknown message type",
		},
		{
			name: "invalid DATA message - mismatched arrays",
			msg: WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeData,
				},
				Payload: DataMessage{
					SeriesID: 0,
					Length:   2,
					X:        []float64{1.0, 2.0},
					Y:        []float64{1.0}, // Mismatched length
				},
			},
			errContains: "must have same length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeWSMessage(tt.msg)
			if tt.errContains == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// TestHeaderAlignment verifies the header is exactly 8 bytes as documented
func TestHeaderAlignment(t *testing.T) {
	if EnvelopeHeaderSize != 8 {
		t.Errorf("HeaderSize = %d, want 8 (must be aligned)", EnvelopeHeaderSize)
	}

	// Verify encoded header is exactly 8 bytes
	env := EnvelopeHeader{
		Version: ProtocolVersion,
		Type:    MessageTypeData,
		Length:  0,
	}
	encoded := EncodeEnvelopeHeader(env)
	if len(encoded) != 8 {
		t.Errorf("encoded header length = %d, want 8", len(encoded))
	}
}

// TestReservedBytesIgnored verifies that reserved bytes can be any value (forward compatibility)
func TestReservedBytesIgnored(t *testing.T) {
	tests := []struct {
		name     string
		reserved [2]byte
	}{
		{"zero bytes", [2]byte{0x00, 0x00}},
		{"arbitrary values", [2]byte{0xAB, 0xCD}},
		{"all ones", [2]byte{0xFF, 0xFF}},
		{"mixed", [2]byte{0x12, 0x34}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := EnvelopeHeader{
				Version:  ProtocolVersion,
				Reserved: tt.reserved,
				Type:     MessageTypeData,
				Length:   100,
			}

			encoded := EncodeEnvelopeHeader(env)
			decoded, err := DecodeEnvelopeHeader(encoded)
			if err != nil {
				t.Fatalf("DecodeEnvelopeHeader() error = %v", err)
			}

			// Reserved bytes should be preserved
			if decoded.Reserved != tt.reserved {
				t.Errorf("Reserved = %v, want %v", decoded.Reserved, tt.reserved)
			}
		})
	}
}

// TestByteOrder verifies Little Endian byte order
func TestByteOrder(t *testing.T) {
	// Test uint32 encoding
	env := EnvelopeHeader{
		Version: ProtocolVersion,
		Type:    MessageTypeData,
		Length:  0x12345678, // Test value with distinct bytes
	}

	encoded := EncodeEnvelopeHeader(env)

	// Verify Little Endian: least significant byte first
	// Length is at bytes 4-7
	if encoded[4] != 0x78 || encoded[5] != 0x56 || encoded[6] != 0x34 || encoded[7] != 0x12 {
		t.Errorf("Length not in Little Endian format: got %02x %02x %02x %02x", encoded[4], encoded[5], encoded[6], encoded[7])
	}
}

// TestFloat64Encoding verifies float64 encoding accuracy
func TestFloat64Encoding(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"zero", 0.0},
		{"negative zero", math.Copysign(0, -1)},
		{"one", 1.0},
		{"pi", math.Pi},
		{"e", math.E},
		{"max float64", math.MaxFloat64},
		{"min float64", -math.MaxFloat64},
		{"smallest nonzero", math.SmallestNonzeroFloat64},
		{"positive infinity", math.Inf(1)},
		{"negative infinity", math.Inf(-1)},
		{"NaN", math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := DataMessage{
				SeriesID: 0,
				Length:   1,
				X:        []float64{tt.value},
				Y:        []float64{tt.value},
			}

			encoded, err := EncodeDataMessage(msg)
			if err != nil {
				t.Fatalf("EncodeDataMessage() error = %v", err)
			}

			decoded, err := DecodeDataMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeDataMessage() error = %v", err)
			}

			if !floatEqual(decoded.X[0], tt.value) {
				t.Errorf("X[0] = %v, want %v", decoded.X[0], tt.value)
			}
			if !floatEqual(decoded.Y[0], tt.value) {
				t.Errorf("Y[0] = %v, want %v", decoded.Y[0], tt.value)
			}
		})
	}
}

// Helper functions

// makeSampleData creates a slice of sample float64 data
func makeSampleData(n int) []float64 {
	data := make([]float64, n)
	for i := 0; i < n; i++ {
		data[i] = float64(i) * 1.5
	}
	return data
}

// floatEqual compares two float64 values, handling NaN and Inf correctly
func floatEqual(a, b float64) bool {
	// Handle NaN: NaN != NaN, so we need special handling
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	// Handle Inf
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return true
	}
	if math.IsInf(a, -1) && math.IsInf(b, -1) {
		return true
	}
	// Regular comparison
	return a == b
}

// floatPtr returns a pointer to a float64 value
func floatPtr(f float64) *float64 {
	return &f
}
