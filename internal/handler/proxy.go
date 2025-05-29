package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/foxinuni/distribuidos-proxy/internal/services"
	"github.com/go-zeromq/zmq4"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Address  string
	Weight   int
	Alive    bool
	LastPong time.Time
	Mutex    sync.Mutex
}

type Connection struct {
	Address string
	Dealer  zmq4.Socket
}

type Proxy struct {
	// internal
	port      int
	workers   int
	heartbeat time.Duration
	deathtime time.Duration
	waitgroup sync.WaitGroup

	stopch   chan struct{}
	requests chan [][]byte

	servers     map[string]*Server
	connections map[string]Connection

	// external
	socket     zmq4.Socket
	serializer services.ModelSerializer
}

func NewServer(
	serializer services.ModelSerializer,
	options ...ProxyOptions,
) *Proxy {
	server := &Proxy{
		port:        5555,
		heartbeat:   250 * time.Millisecond,
		deathtime:   1 * time.Second,
		workers:     10,
		requests:    make(chan [][]byte),
		stopch:      make(chan struct{}),
		servers:     make(map[string]*Server),
		connections: make(map[string]Connection),
		serializer:  serializer,
	}

	for _, applyOption := range options {
		applyOption(server)
	}

	server.registerServers()

	return server
}

func (s *Proxy) Start() error {
	log.Info().Msgf("Starting server on port %d with %d workers", s.port, s.workers)

	// Start the socket
	/*
		s.socket = goczmq.NewRouterChanneler(fmt.Sprintf("tcp://*:%d", s.port))
		if s.socket == nil {
			return fmt.Errorf("failed to create socket")
		}
	*/
	s.socket = zmq4.NewRouter(context.Background(),
		zmq4.WithTimeout(1000*time.Millisecond),
	)

	defer s.socket.Close()

	if err := s.socket.Listen(fmt.Sprintf("tcp://*:%d", s.port)); err != nil {
		return fmt.Errorf("failed to bind socket: %w", err)
	}

	// Start the workers
	for i := range s.workers {
		s.waitgroup.Add(1)

		go func() {
			defer s.waitgroup.Done()
			s.worker(i + 1)
		}()
	}

	// Start the main loop
	go func() {
		defer close(s.requests)

		log.Info().Msg("Starting main loop for server ...")
		for {
			select {
			case <-s.stopch:
				log.Warn().Msg("Stop signal received, exiting main loop")
				return
			default:
				message, err := s.socket.Recv()
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						log.Error().Err(err).Msg("Failed to receive message from socket")
					}

					continue
				}

				s.requests <- message.Frames
			}
		}
	}()

	return nil
}

func (s *Proxy) Stop() {
	log.Info().Msg("Initiating shutdown sequence for server ...")

	// Send stop signal to the main loop
	s.stopch <- struct{}{}
	close(s.stopch)

	// Wait for all workers to finish
	s.waitgroup.Wait()

	log.Info().Msg("Server shutdown complete.")
}

func (s *Proxy) worker(number int) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Panic recovered in worker %d: %v", number, r)
		}
	}()

	for request := range s.requests {
		log.Debug().Msgf("Received request from client (worker: %d, size: %d, identity: %v)", number, len(request[1]), request[0])

		// Parse the request
		req, identity, err := s.parseRequest(request)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse request")

			// Send error encoded
			encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("invalid request format: %w", err))
			s.socket.Send(zmq4.NewMsgFrom(encoded...))
			continue
		}

		dealer, err := s.getDealerForConnection(identity)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get dealer for connection")

			// Send error encoded
			encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("failed to get dealer for connection: %w", err))
			s.socket.Send(zmq4.NewMsgFrom(encoded...))
			continue
		}

		// Forward the request to the dealer
		if err := dealer.Send(zmq4.NewMsgFrom(request[1])); err != nil {
			log.Error().Err(err).Msgf("Failed to send request to dealer for identity %s", identity)
			// Send error encoded
			encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("failed to send request to dealer: %w", err))
			s.socket.Send(zmq4.NewMsgFrom(encoded...))
			continue
		}

		// Wait for the response from the dealer
		response, err := dealer.Recv()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to receive response from dealer for identity %s", identity)
			// Send error encoded
			encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("failed to receive response from dealer: %w", err))
			s.socket.Send(zmq4.NewMsgFrom(encoded...))
			continue
		}

		log.Debug().Msgf("Received response from dealer (worker: %d, identity: %s, type: %s, id: %d)", number, identity, req.Type, req.ID)

		// Send the response back to the client
		s.socket.Send(zmq4.NewMsgFrom([][]byte{[]byte(identity), response.Frames[0]}...))
	}
}
