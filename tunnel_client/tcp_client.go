package tunnel_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type TunnelClient struct {
	ID       string
	ClientID string
	URL      string
}

type TunnelClientOptions struct {
	ID       string
	ClientID string
	URL      string
}

func NewTunnelClient(opts TunnelClientOptions) *TunnelClient {
	return &TunnelClient{ID: opts.ID, ClientID: opts.ClientID, URL: opts.URL}
}

func (tc *TunnelClient) Connect() {
	for {
		log.Println("Attempting to connect to server...")
		conn, err := net.Dial("tcp", tc.URL)
		if err != nil {
			log.Printf("Failed to connect: %v. Retrying in 3 seconds...", err)
			time.Sleep(3 * time.Second)
			continue
		}

		if err := tc.handleConnection(conn); err != nil {
			log.Printf("Connection error: %v. Will reconnect...", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (tc *TunnelClient) handleConnection(conn net.Conn) error {
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	if err := encoder.Encode(map[string]string{"client_id": tc.ClientID, "id": tc.ID}); err != nil {
		return fmt.Errorf("failed to send client ID: %v", err)
	}

	// Receive client ID
	var initResp map[string]string
	if err := decoder.Decode(&initResp); err != nil {
		return fmt.Errorf("failed to decode init response: %v", err)
	}
	log.Printf("Connected with ID: %s", initResp["client_id"])

	// Handle requests
	for {
		var req TunnelRequest
		if err := decoder.Decode(&req); err != nil {
			return fmt.Errorf("error reading request: %v", err)
		}

		// Handle the request
		resp := tc.handleRequest(&req)

		// Send response back
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("error sending response: %v", err)
		}
	}
}

func (tc *TunnelClient) handleRequest(req *TunnelRequest) *TunnelResponse {
	// Create a new HTTP request
	httpReq, err := http.NewRequest(req.Method, req.Target+req.Path, bytes.NewBuffer(req.Body))
	if err != nil {
		return &TunnelResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: []byte(`{"error": "Failed to create request"}`),
		}
	}

	// Copy headers from tunnel request to HTTP request
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return &TunnelResponse{
			StatusCode: http.StatusBadGateway,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: []byte(`{"error": "Failed to forward request"}`),
		}
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TunnelResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: []byte(`{"error": "Failed to read response"}`),
		}
	}

	// Copy headers from HTTP response
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	headers["X-TUNNEL-CLIENT"] = tc.ID
	return &TunnelResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}
}
