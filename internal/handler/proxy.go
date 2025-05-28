package handler

import (
	"fmt"
	"sync"
	"time"

	"github.com/foxinuni/distribuidos-proxy/internal/services"
	"github.com/rs/zerolog/log"
	"gopkg.in/zeromq/goczmq.v4"
)

type Server struct {
	Address  string
	Weight   int
	Alive    bool
	LastPong time.Time
	Mutex    sync.Mutex
}

type Connection struct {
	Dealer *goczmq.Channeler
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
	connections map[string]int

	// external
	socket     *goczmq.Channeler
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
		connections: make(map[string]int),
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
	s.socket = goczmq.NewRouterChanneler(fmt.Sprintf("tcp://*:%d", s.port))
	if s.socket == nil {
		return fmt.Errorf("failed to create socket")
	}

	// Start the workers
	for i := 0; i < s.workers; i++ {
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
			case request := <-s.socket.RecvChan:
				// Check if the channel is closed
				// log.Debug().Msgf("Received request from client: %v", request)
				s.requests <- request
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

	// Shutdown the socket
	if s.socket != nil {
		s.socket.Destroy()
	}

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
			s.socket.SendChan <- encoded
			continue
		}

		encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("proxy is currently not processing requests"))
		s.socket.SendChan <- encoded

		/*
			// Process the request
			response, err := s.processRequest(req)
			if err != nil {
				log.Error().Err(err).Msg("Failed to process request")

				// Send error encoded
				encoded := s.generateErrorResponse(identity, req.ID, req.Type, fmt.Errorf("failed to process request: %w", err))
				s.socket.SendChan <- encoded
				continue
			}

			// Send the response
			encoded := s.generateSuccessResponse(identity, req.ID, req.Type, response)
			s.socket.SendChan <- encoded
		*/
	}
}
