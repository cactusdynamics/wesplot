package wesplot

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const bufferSize = 10000
const maxBufferItemCapacity = 25000

type StreamEndedMessage struct {
	StreamEnded bool   `json:"StreamEnded"`
	StreamError string `json:"StreamError"`
}

type HttpServer struct {
	dataBroadcaster *DataBroadcaster
	host            string
	port            uint16
	metadata        Metadata
	flushInterval   time.Duration
	mux             *http.ServeMux
	logger          *slog.Logger
}

func NewHttpServer(dataBroadcaster *DataBroadcaster, host string, port uint16, metadata Metadata, flushInterval time.Duration) *HttpServer {

	s := &HttpServer{
		dataBroadcaster: dataBroadcaster,
		host:            host,
		port:            port,
		metadata:        metadata,
		flushInterval:   flushInterval,
		mux:             http.NewServeMux(),
		logger:          slog.Default().With("tag", "HttpServer"),
	}

	subFS, err := fs.Sub(webuiFiles, "webui")
	if err != nil {
		panic(err)
	}

	s.mux.Handle("/", http.FileServer(http.FS(subFS)))
	s.mux.HandleFunc("/ws", s.handleWebSocket)
	s.mux.HandleFunc("/ws2", s.handleWebSocket2)
	s.mux.HandleFunc("/metadata", s.handleMetadata)
	s.mux.HandleFunc("/errors", s.handleErrors)

	return s
}

func (s *HttpServer) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	// TODO: need to ensure that we allow CORS.
	c, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.logger.With("error", err).Warn("failed to accept new websocket connection")
		return
	}

	ctx := req.Context()
	ctx = c.CloseRead(ctx) // This means we no longer want to read from the websocket, which is true because we just want to write.

	channel := make(chan DataRow, bufferSize)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		// We buffer data for at least X milliseconds or if it reaches capacity before sending it to the client.
		// Note: tune or allow configuration
		bufferItemCapacity := Min(s.metadata.WindowSize, maxBufferItemCapacity)
		lastSendTime := time.Now()
		dataBuffer := make([]DataRow, 0, bufferItemCapacity)

		flushBufferToWebsocket := func() error {
			err := wsjson.Write(ctx, c, dataBuffer)
			if err != nil {
				return err
			}

			dataBuffer = make([]DataRow, 0, bufferItemCapacity) // TODO: try to clear the buffer without allocating
			lastSendTime = time.Now()
			return nil
		}

		logger := s.logger.With("channel", channel)

		for {
			select {
			case dataRow, open := <-channel:
				if !open {
					// Not sure why this would ever happen, but sure
					// TODO: maybe panic here
					logger.Warn("data channel closed, closing websocket")
					c.Close(websocket.StatusNormalClosure, "channel closed")
					return
				}

				if dataRow.streamEnded {
					// Stream has ended. We should close the the websocket. The client
					// should issue another request to /errors after the websocket
					// connection closes to see if there are any stream errors so it can
					// display it.
					logger.Info("stream ended, flushing and then closing websocket connection")
					err := flushBufferToWebsocket()
					if err != nil {
						logger.Warn("websocket flush failed and closed")
						return
					}

					c.Close(websocket.StatusNormalClosure, "")
					return
				}

				dataBuffer = append(dataBuffer, dataRow)
				if len(dataBuffer) >= bufferItemCapacity || time.Since(lastSendTime) > s.flushInterval {
					logger.With("buflen", len(dataBuffer)).Debug("buffer capacity reached, flushing")
					err := flushBufferToWebsocket()
					if err != nil {
						// At this point the websocket closed, so we don't even need to send anything
						logger.Warn("websocket write failed and closed")
						return
					}
				}

			case <-time.After(s.flushInterval):
				if len(dataBuffer) > 0 {
					logger.With("buflen", len(dataBuffer)).Debug("timed out waiting for more data, flushing")
					err := flushBufferToWebsocket()
					if err != nil {
						// At this point the websocket closed, so we don't even need to send anything
						logger.Warn("websocket write failed and closed")
						return
					}
				}

			case <-ctx.Done(): // client connection closes causes the req.Context to be canceled?
				logger.Info("client closed connection or context canceled")
				c.Close(websocket.StatusNormalClosure, "")
				return
			}
		}
	}()

	// The channel is already being received from in another goroutine and we
	// register the channels in the main thread.
	s.dataBroadcaster.RegisterChannel(ctx, channel)

	// Once the websocket writing thread finishes, we want to deregister the
	// channel from the broadcaster.
	wg.Wait()
	s.dataBroadcaster.DeregisterChannel(ctx, channel)
	close(channel)
}

func (s *HttpServer) handleWebSocket2(w http.ResponseWriter, req *http.Request) {
	// Accept websocket connection with binary frame support
	c, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.logger.With("error", err).Warn("failed to accept new websocket connection")
		return
	}

	ctx := req.Context()
	ctx = c.CloseRead(ctx) // This means we no longer want to read from the websocket

	channel := make(chan DataRow, bufferSize)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		logger := s.logger.With("channel", channel, "endpoint", "/ws2")

		// Send metadata message immediately on connection
		metadataMsg := WSMessage{
			Header: EnvelopeHeader{
				Version: ProtocolVersion,
				Type:    MessageTypeMetadata,
			},
			Payload: s.metadata,
		}

		metadataBytes, err := EncodeWSMessage(metadataMsg)
		if err != nil {
			logger.With("error", err).Error("failed to encode metadata message")
			c.Close(websocket.StatusInternalError, "metadata encoding failed")
			return
		}

		err = c.Write(ctx, websocket.MessageBinary, metadataBytes)
		if err != nil {
			logger.With("error", err).Warn("failed to send metadata message")
			return
		}

		// Buffer data for at least X milliseconds or if it reaches capacity before sending
		bufferItemCapacity := Min(s.metadata.WindowSize, maxBufferItemCapacity)

		// Pre-allocate X/Y buffers for each series to avoid reallocation
		numSeries := len(s.metadata.WesplotOptions.Columns)
		xBuffers := make(map[int][]float64, numSeries)
		yBuffers := make(map[int][]float64, numSeries)
		lastSendTimes := make(map[int]time.Time)
		now := time.Now()
		for i := 0; i < numSeries; i++ {
			xBuffers[i] = make([]float64, 0, bufferItemCapacity)
			yBuffers[i] = make([]float64, 0, bufferItemCapacity)
			lastSendTimes[i] = now
		}

		// flushSeries flushes a single series
		flushSeries := func(seriesID int) error {
			if len(xBuffers[seriesID]) == 0 {
				return nil
			}

			dataMsg := DataMessage{
				SeriesID: uint32(seriesID),
				Length:   uint32(len(xBuffers[seriesID])),
				X:        xBuffers[seriesID],
				Y:        yBuffers[seriesID],
			}

			wsMsg := WSMessage{
				Header: EnvelopeHeader{
					Version: ProtocolVersion,
					Type:    MessageTypeData,
				},
				Payload: dataMsg,
			}

			dataBytes, err := EncodeWSMessage(wsMsg)
			if err != nil {
				return fmt.Errorf("failed to encode data message for series %d: %w", seriesID, err)
			}

			err = c.Write(ctx, websocket.MessageBinary, dataBytes)
			if err != nil {
				return fmt.Errorf("failed to write data message for series %d: %w", seriesID, err)
			}

			// Clear buffer by resetting length (reuse capacity)
			xBuffers[seriesID] = xBuffers[seriesID][:0]
			yBuffers[seriesID] = yBuffers[seriesID][:0]
			lastSendTimes[seriesID] = time.Now()
			return nil
		}

		for {
			select {
			case dataRow, open := <-channel:
				if !open {
					logger.Warn("data channel closed, closing websocket")
					c.Close(websocket.StatusNormalClosure, "channel closed")
					return
				}

				if dataRow.streamEnded {
					// Stream has ended. Flush buffered data first, then send stream end message
					logger.Info("stream ended, flushing and then closing websocket connection")

					for seriesID := 0; seriesID < numSeries; seriesID++ {
						if err := flushSeries(seriesID); err != nil {
							logger.With("error", err).Warn("websocket flush failed")
							return
						}
					}

					// Send stream end envelope
					streamEndMsg := StreamEndMessage{
						Error: dataRow.streamErr != nil,
						Msg:   "",
					}
					if dataRow.streamErr != nil {
						streamEndMsg.Msg = dataRow.streamErr.Error()
					}

					wsMsg := WSMessage{
						Header: EnvelopeHeader{
							Version: ProtocolVersion,
							Type:    MessageTypeStreamEnd,
						},
						Payload: streamEndMsg,
					}

					streamEndBytes, err := EncodeWSMessage(wsMsg)
					if err != nil {
						logger.With("error", err).Error("failed to encode stream end message")
					} else {
						err = c.Write(ctx, websocket.MessageBinary, streamEndBytes)
						if err != nil {
							logger.With("error", err).Warn("failed to send stream end message")
						}
					}

					c.Close(websocket.StatusNormalClosure, "")
					return
				}

				// Immediately transform DataRow into X/Y arrays for each series
				for seriesID := 0; seriesID < numSeries; seriesID++ {
					xBuffers[seriesID] = append(xBuffers[seriesID], dataRow.X)
					yBuffers[seriesID] = append(yBuffers[seriesID], dataRow.Ys[seriesID])

					// Flush this series if it reaches capacity or timeout
					if len(xBuffers[seriesID]) >= bufferItemCapacity || time.Since(lastSendTimes[seriesID]) > s.flushInterval {
						logger.With("seriesID", seriesID, "buflen", len(xBuffers[seriesID])).Debug("series buffer capacity reached, flushing")
						err := flushSeries(seriesID)
						if err != nil {
							logger.With("error", err, "seriesID", seriesID).Warn("websocket write failed and closed")
							return
						}
					}
				}

			case <-time.After(s.flushInterval):
				// Check each series independently for timeout flush
				for seriesID := 0; seriesID < numSeries; seriesID++ {
					logger.With("seriesID", seriesID, "buflen", len(xBuffers[seriesID])).Debug("timed out waiting for more data, flushing series")
					err := flushSeries(seriesID)
					if err != nil {
						logger.With("error", err, "seriesID", seriesID).Warn("websocket write failed and closed")
						return
					}
				}

			case <-ctx.Done():
				logger.Info("client closed connection or context canceled")
				c.Close(websocket.StatusNormalClosure, "")
				return
			}
		}
	}()

	// Register the channel with the broadcaster
	s.dataBroadcaster.RegisterChannel(ctx, channel)

	// Once the websocket writing thread finishes, deregister the channel
	wg.Wait()
	s.dataBroadcaster.DeregisterChannel(ctx, channel)
	close(channel)
}

func (s *HttpServer) handleMetadata(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "content-type")
	w.Header().Add("Access-Control-Allow-Methods", "*")
	err := json.NewEncoder(w).Encode(s.metadata)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (s *HttpServer) handleErrors(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "content-type")
	w.Header().Add("Access-Control-Allow-Methods", "*")

	streamEnded := s.dataBroadcaster.streamEnded.Load()
	var streamEndedMessage StreamEndedMessage
	if streamEnded {
		streamEndedMessage.StreamEnded = true
		if s.dataBroadcaster.err != nil {
			streamEndedMessage.StreamError = s.dataBroadcaster.err.Error()
		} else {
			streamEndedMessage.StreamError = ""
		}
	}

	err := json.NewEncoder(w).Encode(streamEndedMessage)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (s *HttpServer) Run() error {
	tries := 0
	var addr string
	var listener net.Listener
	var err error

	for {
		if tries > 200 {
			panic("tried 200 ports and they all failed?") // Not sure if this is needed
		}

		addr = fmt.Sprintf("%s:%d", s.host, s.port)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			s.port++
			tries++
			// Really should try to distinguish which error is an address bind error.
			// However not sure how to do this in a cross platform manner.
			// TODO: fix me.
			s.logger.With(
				"error", err,
				"addr", addr,
				"nextHost", s.host,
				"nextPort", s.port,
			).Warn("failed to listen on address, trying next port")
		} else {
			break
		}
	}

	// These log lines don't need to be tagged (as that introduces more confusion)
	url := fmt.Sprintf("http://%s:%d", s.host, s.port)
	openBrowser(url)

	if s.host == "0.0.0.0" {
		ifaces, err := net.Interfaces()
		if err != nil {
			panic(fmt.Sprintf("cannot get network interfaces: %v", err))
		}

		slog.Info("Plot is accessible at all IP addresses (IPv4 shown below)")
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				panic(fmt.Sprintf("cannot get iface addr: %v", err))
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				ipv4 := ip.To4()
				if ipv4 != nil {
					slog.Info("http endpoint", "url", fmt.Sprintf("http://%s:%d", ipv4, s.port))
				}
			}

		}
	} else {
		slog.Info("Plot is accessible", "url", url)
	}

	server := http.Server{Addr: addr, Handler: s.mux}
	return server.Serve(listener)
}
