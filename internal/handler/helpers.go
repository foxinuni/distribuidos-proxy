package handler

import (
	"fmt"
	"math/rand"

	"github.com/foxinuni/distribuidos-proxy/internal/models"
	"github.com/rs/zerolog/log"
)

func (s *Proxy) parseRequest(request [][]byte) (*models.Request, string, error) {
	if len(request) < 2 {
		return nil, "", fmt.Errorf("invalid request format")
	}

	// Deserialize the request
	var req models.Request
	if err := s.serializer.Decode(request[1], &req); err != nil {
		return nil, "", fmt.Errorf("failed to decode request: %w", err)
	}

	// Get the sender ID
	identity := string(request[0])

	return &req, identity, nil
}

func (s *Proxy) parseResponse(response [][]byte) (*models.Response, error) {
	// Deserialize the response
	var resp models.Response
	if err := s.serializer.Decode(response[0], &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func (s *Proxy) generateErrorResponse(identity string, id int, handler string, err error) [][]byte {
	response := &models.Response{
		ID:      id,
		Type:    handler,
		Success: false,
		Error:   err.Error(),
	}

	// Serialize the response
	encoded, err := s.serializer.Encode(response)
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize error response")
		return nil
	}

	// Send the response
	return [][]byte{[]byte(identity), encoded}
}

func (s *Proxy) generateSuccessResponse(identity string, id int, handler string, content interface{}) [][]byte {
	response := &models.Response{
		ID:      id,
		Type:    handler,
		Success: true,
		Content: content,
	}

	// Serialize the response
	encoded, err := s.serializer.Encode(response)
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize success response")
		return nil
	}

	// Send the response
	return [][]byte{[]byte(identity), encoded}
}

func (s *Proxy) generateHeartbeat() []byte {
	heartbeat := &models.Request{
		ID:   rand.Intn(1000000), // Random ID for heartbeat
		Type: "health-check",
	}

	// Serialize the heartbeat
	encoded, err := s.serializer.Encode(heartbeat)
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize heartbeat")
		return nil
	}

	// Send the heartbeat
	return encoded
}
