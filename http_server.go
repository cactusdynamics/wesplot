package wesplot

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const bufferSize = 10000

type StreamEndedMessage struct {
	StreamEnded bool
	StreamError error
}

type HttpServer struct {
	dataBroadcaster *DataBroadcaster
	host            string
	port            uint16
	metadata        Metadata
	mux             *http.ServeMux
	logger          logrus.FieldLogger
}

func NewHttpServer(dataBroadcaster *DataBroadcaster, host string, port uint16, metadata Metadata) *HttpServer {

	s := &HttpServer{
		dataBroadcaster: dataBroadcaster,
		host:            host,
		port:            port,
		metadata:        metadata,
		mux:             http.NewServeMux(),
		logger:          logrus.WithField("tag", "HttpServer"),
	}

	subFS, err := fs.Sub(webuiFiles, "webui")
	if err != nil {
		panic(err)
	}

	s.mux.Handle("/", http.FileServer(http.FS(subFS)))
	s.mux.HandleFunc("/ws", s.handleWebSocket)
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
		s.logger.WithError(err).Warn("failed to accept new websocket connection")
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
		const bufferTimeCapacity = 50 * time.Millisecond
		bufferItemCapacity := Min(s.metadata.WindowSize, 25000)
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

		logger := s.logger.WithField("channel", channel)

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
				if len(dataBuffer) >= bufferItemCapacity || time.Since(lastSendTime) > bufferTimeCapacity {
					logger.WithField("buflen", len(dataBuffer)).Debug("buffer capacity reached, flushing")
					err := flushBufferToWebsocket()
					if err != nil {
						// At this point the websocket closed, so we don't even need to send anything
						logger.Warn("websocket write failed and closed")
						return
					}
				}

			case <-time.After(bufferTimeCapacity):
				if len(dataBuffer) > 0 {
					logger.WithField("buflen", len(dataBuffer)).Debug("timed out waiting for more data, flushing")
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
		streamEndedMessage.StreamError = s.dataBroadcaster.err
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
			s.logger.WithError(err).Warnf("failed to listen on %s, trying %s:%d instead", addr, s.host, s.port)
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

		logrus.Info("Plot is accessible at all IP addresses (IPv4 shown below):")
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
					logrus.Infof("  - http://%s:%d", ipv4, s.port)
				}
			}

		}
	} else {
		logrus.Info("Plot is accessible at: %s", url)
	}

	server := http.Server{Addr: addr, Handler: s.mux}
	return server.Serve(listener)
}
