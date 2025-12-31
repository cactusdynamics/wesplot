package wesplot

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// Protocol constants
const (
	// ProtocolVersion is the current version of the WS2 protocol
	ProtocolVersion byte = 1

	// Message type constants
	MessageTypeData      byte = 0x01
	MessageTypeMetadata  byte = 0x02
	MessageTypeStreamEnd byte = 0x03

	// Header size in bytes
	EnvelopeHeaderSize = 8
)

// EnvelopeHeader represents the message envelope header
type EnvelopeHeader struct {
	Version  byte
	Reserved [2]byte // Reserved for future use
	Type     byte
	Length   uint32 // Payload length in bytes
}

// DataMessage represents a DATA message payload (type 0x01)
type DataMessage struct {
	SeriesID uint32
	Length   uint32    // Number of X/Y pairs
	X        []float64 // X values
	Y        []float64 // Y values
}

// StreamEndMessage represents a STREAM_END message payload (type 0x03)
type StreamEndMessage struct {
	Error bool
	Msg   string
}

// WSMessage represents a complete websocket message with header and payload
type WSMessage struct {
	Header  EnvelopeHeader
	Payload interface{} // One of: DataMessage, Metadata, StreamEndMessage
}

// EncodeEnvelopeHeader encodes the envelope header into a byte slice
func EncodeEnvelopeHeader(env EnvelopeHeader) []byte {
	buf := make([]byte, EnvelopeHeaderSize)
	buf[0] = env.Version
	buf[1] = env.Reserved[0]
	buf[2] = env.Reserved[1]
	buf[3] = env.Type
	binary.LittleEndian.PutUint32(buf[4:8], env.Length)
	return buf
}

// DecodeEnvelopeHeader decodes the envelope header from a byte slice
// Returns the envelope and an error if the buffer is too short
func DecodeEnvelopeHeader(buf []byte) (EnvelopeHeader, error) {
	if len(buf) < EnvelopeHeaderSize {
		return EnvelopeHeader{}, fmt.Errorf("buffer too short: expected at least %d bytes, got %d", EnvelopeHeaderSize, len(buf))
	}

	env := EnvelopeHeader{
		Version: buf[0],
		Type:    buf[3],
		Length:  binary.LittleEndian.Uint32(buf[4:8]),
	}
	env.Reserved[0] = buf[1]
	env.Reserved[1] = buf[2]

	return env, nil
}

// EncodeDataMessage encodes a DATA message payload
// Returns error if X and Y arrays don't match in length
func EncodeDataMessage(msg DataMessage) ([]byte, error) {
	if len(msg.X) != len(msg.Y) {
		return nil, fmt.Errorf("X and Y arrays must have same length: X=%d, Y=%d", len(msg.X), len(msg.Y))
	}
	if uint32(len(msg.X)) != msg.Length {
		return nil, fmt.Errorf("Length field (%d) doesn't match array length (%d)", msg.Length, len(msg.X))
	}

	// Calculate payload size: SeriesID(4) + Length(4) + X array + Y array
	payloadSize := 8 + (msg.Length * 8 * 2)
	buf := make([]byte, payloadSize)

	// Encode SeriesID and Length
	binary.LittleEndian.PutUint32(buf[0:4], msg.SeriesID)
	binary.LittleEndian.PutUint32(buf[4:8], msg.Length)

	// Encode X array
	offset := 8
	for _, x := range msg.X {
		binary.LittleEndian.PutUint64(buf[offset:offset+8], math.Float64bits(x))
		offset += 8
	}

	// Encode Y array
	for _, y := range msg.Y {
		binary.LittleEndian.PutUint64(buf[offset:offset+8], math.Float64bits(y))
		offset += 8
	}

	return buf, nil
}

// DecodeDataMessage decodes a DATA message payload
func DecodeDataMessage(buf []byte) (DataMessage, error) {
	if len(buf) < 8 {
		return DataMessage{}, fmt.Errorf("buffer too short for DATA message: expected at least 8 bytes, got %d", len(buf))
	}

	msg := DataMessage{
		SeriesID: binary.LittleEndian.Uint32(buf[0:4]),
		Length:   binary.LittleEndian.Uint32(buf[4:8]),
	}

	// Validate buffer size
	expectedSize := 8 + (msg.Length * 8 * 2)
	if uint32(len(buf)) != expectedSize {
		return DataMessage{}, fmt.Errorf("buffer size mismatch: expected %d bytes for %d pairs, got %d", expectedSize, msg.Length, len(buf))
	}

	// Decode X array
	msg.X = make([]float64, msg.Length)
	offset := 8
	for i := uint32(0); i < msg.Length; i++ {
		bits := binary.LittleEndian.Uint64(buf[offset : offset+8])
		msg.X[i] = math.Float64frombits(bits)
		offset += 8
	}

	// Decode Y array
	msg.Y = make([]float64, msg.Length)
	for i := uint32(0); i < msg.Length; i++ {
		bits := binary.LittleEndian.Uint64(buf[offset : offset+8])
		msg.Y[i] = math.Float64frombits(bits)
		offset += 8
	}

	return msg, nil
}

// EncodeMetadataMessage encodes a METADATA message payload
// Takes the Metadata struct and returns the encoded payload
func EncodeMetadataMessage(metadata Metadata) ([]byte, error) {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Payload: JSON Length (4 bytes) + JSON data
	payloadSize := 4 + len(jsonData)
	buf := make([]byte, payloadSize)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(jsonData)))
	copy(buf[4:], jsonData)

	return buf, nil
}

// DecodeMetadataMessage decodes a METADATA message payload
func DecodeMetadataMessage(buf []byte) (Metadata, error) {
	if len(buf) < 4 {
		return Metadata{}, fmt.Errorf("buffer too short for METADATA message: expected at least 4 bytes, got %d", len(buf))
	}

	jsonLength := binary.LittleEndian.Uint32(buf[0:4])

	// Validate buffer size
	expectedSize := 4 + jsonLength
	if uint32(len(buf)) != expectedSize {
		return Metadata{}, fmt.Errorf("buffer size mismatch: expected %d bytes, got %d", expectedSize, len(buf))
	}

	var metadata Metadata
	if err := json.Unmarshal(buf[4:], &metadata); err != nil {
		return Metadata{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}

// EncodeStreamEndMessage encodes a STREAM_END message payload
func EncodeStreamEndMessage(msg StreamEndMessage) ([]byte, error) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream end message: %w", err)
	}

	// Payload: JSON Length (4 bytes) + JSON data
	payloadSize := 4 + len(jsonData)
	buf := make([]byte, payloadSize)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(jsonData)))
	copy(buf[4:], jsonData)

	return buf, nil
}

// DecodeStreamEndMessage decodes a STREAM_END message payload
func DecodeStreamEndMessage(buf []byte) (StreamEndMessage, error) {
	if len(buf) < 4 {
		return StreamEndMessage{}, fmt.Errorf("buffer too short for STREAM_END message: expected at least 4 bytes, got %d", len(buf))
	}

	jsonLength := binary.LittleEndian.Uint32(buf[0:4])

	// Validate buffer size
	expectedSize := 4 + jsonLength
	if uint32(len(buf)) != expectedSize {
		return StreamEndMessage{}, fmt.Errorf("buffer size mismatch: expected %d bytes, got %d", expectedSize, len(buf))
	}

	var msg StreamEndMessage
	if err := json.Unmarshal(buf[4:], &msg); err != nil {
		return StreamEndMessage{}, fmt.Errorf("failed to unmarshal stream end message: %w", err)
	}

	return msg, nil
}

// EncodeWSMessage encodes a WSMessage into a complete message byte slice
// Returns error if payload encoding fails or if payload type is invalid
func EncodeWSMessage(msg WSMessage) ([]byte, error) {
	var payload []byte
	var err error

	// Encode payload based on message type
	switch msg.Header.Type {
	case MessageTypeData:
		dataMsg, ok := msg.Payload.(DataMessage)
		if !ok {
			return nil, fmt.Errorf("payload type mismatch: expected DataMessage for type 0x%02x, got %T", msg.Header.Type, msg.Payload)
		}
		payload, err = EncodeDataMessage(dataMsg)
		if err != nil {
			return nil, err
		}
	case MessageTypeMetadata:
		metadata, ok := msg.Payload.(Metadata)
		if !ok {
			return nil, fmt.Errorf("payload type mismatch: expected Metadata for type 0x%02x, got %T", msg.Header.Type, msg.Payload)
		}
		payload, err = EncodeMetadataMessage(metadata)
		if err != nil {
			return nil, err
		}
	case MessageTypeStreamEnd:
		streamEnd, ok := msg.Payload.(StreamEndMessage)
		if !ok {
			return nil, fmt.Errorf("payload type mismatch: expected StreamEndMessage for type 0x%02x, got %T", msg.Header.Type, msg.Payload)
		}
		payload, err = EncodeStreamEndMessage(streamEnd)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown message type: 0x%02x", msg.Header.Type)
	}

	// Update header length to match actual payload size
	msg.Header.Length = uint32(len(payload))

	// Encode header
	header := EncodeEnvelopeHeader(msg.Header)

	// Combine header and payload
	fullMsg := make([]byte, len(header)+len(payload))
	copy(fullMsg, header)
	copy(fullMsg[len(header):], payload)

	return fullMsg, nil
}

// DecodeWSMessage decodes a complete message (envelope + payload) into a WSMessage
// Returns error if buffer is too short or payload decoding fails
func DecodeWSMessage(buf []byte) (WSMessage, error) {
	env, err := DecodeEnvelopeHeader(buf)
	if err != nil {
		return WSMessage{}, err
	}

	// Validate full message size
	expectedSize := EnvelopeHeaderSize + env.Length
	if uint32(len(buf)) < expectedSize {
		return WSMessage{}, fmt.Errorf("buffer too short: expected %d bytes (header + payload), got %d", expectedSize, len(buf))
	}

	payloadBytes := buf[EnvelopeHeaderSize : EnvelopeHeaderSize+env.Length]

	// Decode payload based on message type
	var payload interface{}
	switch env.Type {
	case MessageTypeData:
		dataMsg, err := DecodeDataMessage(payloadBytes)
		if err != nil {
			return WSMessage{}, err
		}
		payload = dataMsg
	case MessageTypeMetadata:
		metadata, err := DecodeMetadataMessage(payloadBytes)
		if err != nil {
			return WSMessage{}, err
		}
		payload = metadata
	case MessageTypeStreamEnd:
		streamEnd, err := DecodeStreamEndMessage(payloadBytes)
		if err != nil {
			return WSMessage{}, err
		}
		payload = streamEnd
	default:
		return WSMessage{}, fmt.Errorf("unknown message type: 0x%02x", env.Type)
	}

	return WSMessage{
		Header:  env,
		Payload: payload,
	}, nil
}
