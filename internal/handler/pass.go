package handler

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/zeromq/goczmq.v4"
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

	// Dealer gets created to connect to the server
	dealer, err := goczmq.NewDealer(server.Address)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create dealer for server %q", server.Address)
		return
	}
	defer dealer.Destroy()

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
				return
			default:
				msg, err := dealer.RecvMessageNoWait()
				if err != nil {
					if !errors.Is(err, goczmq.ErrRecvMessage) {
						log.Error().Err(err).Msgf("Failed to receive message from server %q", server.Address)
						continue
					}

					// log.Info().Msgf("No message received from server %q, retrying...", server.Address)
					time.Sleep(10 * time.Millisecond)
					continue
				}

				log.Info().Msgf("Receiving message from server %q", server.Address)

				if len(msg) < 2 {
					log.Error().Msg("Received invalid message from server")
					continue
				}

				// Parse the request
				res, err := p.parseResponse(msg)
				if err != nil {
					log.Error().Err(err).Msg("Failed to parse request")
					continue
				}

				if res.Type == "health-check" {
					log.Debug().Msgf("Received heartbeat from server: %q", server.Address)

					// Reset the death timer since we received a heartbeat
					deathTimer.Reset(p.deathtime)

					// Update server status
					server.Mutex.Lock()
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
				return
			case <-heartbeatTimer.C:
				request := p.generateHeartbeat()

				log.Info().Msgf("Sending heartbeat to server %q", server.Address)
				if err := dealer.SendMessage([][]byte{request}); err != nil {
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
