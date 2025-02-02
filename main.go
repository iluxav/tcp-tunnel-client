package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	tc "tcp_client/tunnel_client"

	"github.com/google/uuid"
)

func main() {

	tunnelURL := flag.String("url", "64.226.105.234:9091", "URL for the tunnel server")
	clientID := flag.String("client-id", "", "ID for the client")

	flag.Parse()
	if *clientID == "" {
		log.Fatal("Client ID is required '-client-id=my-service'")
	}
	if *tunnelURL == "" {
		log.Fatal("Tunnel URL is required '-url=127.2.3.4:9091'")
	}

	tunnelClient := tc.NewTunnelClient(tc.TunnelClientOptions{
		ID:       uuid.New().String(),
		ClientID: *clientID,
		URL:      *tunnelURL,
	})
	go tunnelClient.Connect()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		responseBody, _ := json.Marshal(map[string]interface{}{
			"message": "securely connected http server!",
			"path":    r.URL.Path,
			"method":  r.Method,
		})

		w.Write(responseBody)
	})
	log.Printf("Starting HTTP server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}

}
