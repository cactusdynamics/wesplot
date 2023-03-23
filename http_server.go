package wesplot

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const bufferSize = 10000

type HttpServer struct {
	dataBroadcaster *DataBroadcaster
	addr            string
	metadata        Metadata
	mux             *http.ServeMux
	logger          logrus.FieldLogger
}

func NewHttpServer(dataBroadcaster *DataBroadcaster, addr string, metadata Metadata) *HttpServer {
	s := &HttpServer{
		dataBroadcaster: dataBroadcaster,
		addr:            addr,
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
		for {
			select {
			case dataRow, open := <-channel:
				if !open { // Not sure why this would ever happen, but sure
					// TODO: maybe panic here
					s.logger.Warn("data channel closed, closing websocket")
					c.Close(websocket.StatusNormalClosure, "channel closed")
					return
				}

				err := wsjson.Write(ctx, c, dataRow)
				if err != nil {
					// At this point the websocket closed, so we don't even need to send anything
					s.logger.Warn("websocket write failed and closed")
					return
				}
			case <-ctx.Done(): // client connection closes causes the req.Context to be canceled?
				s.logger.Info("client closed connection or context canceled")
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
	err := json.NewEncoder(w).Encode(s.metadata)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (s *HttpServer) Run() {
	logrus.Infof("starting HTTP server at http://%s", s.addr)
	http.ListenAndServe(s.addr, s.mux)
}
