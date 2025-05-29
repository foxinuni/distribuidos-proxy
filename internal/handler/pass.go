package handler

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/rs/zerolog/log"
)

func (p *Proxy) registerServers() {
	for _, server := range p.servers {
		log.Info().Msgf("Registering server: %q (weight: %d)", server.Address, server.Weight)

		// Start a goroutine to handle each server
		p.waitgroup.Add(1)
		go func() {
			defer p.waitgroup.Done()
			p.handleServer(server)
		}()
	}
}

func (p *Proxy) handleServer(server *Server) {
	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		
		select {
		case <-p.stopch:
		case <-ctx.Done():
		}
	}()

	// Dealer gets created to connect to the server
	dealer := zmq4.NewDealer(ctx,
		zmq4.WithTimeout(p.deathtime),
		zmq4.WithDialerTimeout(p.deathtime),
		zmq4.WithDialerMaxRetries(-1),
		zmq4.WithAutomaticReconnect(true),
	)
	defer dealer.Close()

	if err := dealer.Dial(server.Address); err != nil {
		log.Error().Err(err).Msgf("Failed to connect to server %q", server.Address)
		return
	}

	// Set the dealer to receive messages
	heartbeatTimer := time.NewTicker(p.heartbeat)
	defer heartbeatTimer.Stop()

	// Death timer to check server health
	deathTimer := time.NewTicker(p.deathtime)
	defer deathTimer.Stop()

	// Start goroutine to handle heartbeat responses and update server status
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-p.stopch:
				log.Debug().Msgf("Stopping heartbeat response server handler for %q", server.Address)
				return
			default:
				// Receive messages from the dealer
				msg, err := dealer.Recv()
				if err != nil {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
						log.Error().Err(err).Msgf("Failed to receive message from server %q", server.Address)
					}

					continue
				}

				// Parse the request
				res, err := p.parseResponse(msg.Frames)
				if err != nil {
					log.Error().Err(err).Msg("Failed to parse request")
					continue
				}

				if res.Type == "health-check" {
					// Reset the death timer since we received a heartbeat
					deathTimer.Reset(p.deathtime)

					// Update server status
					server.Mutex.Lock()

					if !server.Alive {
						log.Info().Msgf("Server %q is now alive (Last Heartbeat: %s)", server.Address, server.LastPong.Format(time.RFC3339))
					}

					server.LastPong = time.Now()
					server.Alive = true
					server.Mutex.Unlock()
					continue
				}
			}
		}
	}()

	// Start a goroutine to periodically send heartbeats and check server health
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-p.stopch:
				log.Debug().Msgf("Stopping heartbeat sender for %q", server.Address)
				return
			case <-heartbeatTimer.C:
				request := p.generateHeartbeat()

				if err := dealer.Send(zmq4.NewMsg(request)); err != nil {
					log.Error().Err(err).Msgf("Failed to send heartbeat to server %q", server.Address)
					continue
				}
			case <-deathTimer.C:
				server.Mutex.Lock()
				if time.Since(server.LastPong) > p.deathtime && server.Alive {
					log.Warn().Msgf("Server %q is considered dead, removing from active servers", server.Address)
					server.Alive = false
				}

				server.Mutex.Unlock()
			}
		}
	}()

	// Start a goroutine to check server health
	wg.Wait()

	log.Warn().Msgf("Server %q handler stopped", server.Address)
}

func (s *Proxy) getDealerForConnection(identity string) (zmq4.Socket, error) {
	// Check if conn exists and server is alive
	if conn, exists := s.connections[identity]; exists {
		// Find the server associated with this identity
		for _, server := range s.servers {
			if server.Address == conn.Address {
				server.Mutex.Lock()
				alive := server.Alive
				server.Mutex.Unlock()

				if alive {
					return conn.Dealer, nil
				}
				break
			}
		}

		// If not alive, close and remove the dealer
		conn.Dealer.Close()
	}

	// Find a server to connect to
	for _, server := range s.servers {
		server.Mutex.Lock()

		if server.Alive {
			dealer := zmq4.NewDealer(context.Background(),
				zmq4.WithTimeout(s.deathtime),
				zmq4.WithDialerTimeout(s.deathtime),
				zmq4.WithDialerMaxRetries(-1),
			)

			if err := dealer.Dial(server.Address); err != nil {
				continue
			}

			s.connections[identity] = Connection{
				Address: server.Address,
				Dealer:  dealer,
			}

			return dealer, nil
		}
	}

	return nil, errors.New("no available servers")
}
