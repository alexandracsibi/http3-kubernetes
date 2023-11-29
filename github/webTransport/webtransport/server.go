package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/adriancable/webtransport-go"
)

func handleWebTransportStreams(session *webtransport.Session) {
	// Handle incoming datagrams
	go func() {
		for {
			msg, err := session.ReceiveMessage(session.Context())
			if err != nil {
				fmt.Println("Session closed, ending datagram listener:", err)
				break
			}
			fmt.Printf("Received datagram: %s\n", msg)

			sendMsg := bytes.ToUpper(msg)
			fmt.Printf("Sending datagram: %s\n", sendMsg)
			session.SendMessage(sendMsg)
		}
	}()

	// Handle incoming unidirectional streams
	go func() {
		for {
			s, err := session.AcceptUniStream(session.Context())
			if err != nil {
				fmt.Println("Session closed, not accepting more unidirectional streams:", err)
				break
			}
			fmt.Println("Accepting incoming unidirectional stream:", s.StreamID())

			go func(s webtransport.ReceiveStream) {
				for {
					buf := make([]byte, 1024)
					n, err := s.Read(buf)
					if err != nil {
						log.Printf("Error reading from unidirectional stream %v: %v\n", s.StreamID(), err)
						break
					}
					fmt.Printf("Received from unidirectional stream: %s\n", buf[:n])
				}
			}(s)
		}
	}()

	// Handle incoming bidirectional streams
	go func() {
		for {
			s, err := session.AcceptStream()
			if err != nil {
				fmt.Println("Session closed, not accepting more bidirectional streams:", err)
				break
			}
			fmt.Println("Accepting incoming bidirectional stream:", s.StreamID())

			go func(s webtransport.Stream) {
				defer s.Close()
				for {
					buf := make([]byte, 1024)
					n, err := s.Read(buf)
					if err != nil {
						log.Printf("Error reading from bidirectional stream %v: %v\n", s.StreamID(), err)
						break
					}
					fmt.Printf("Received from bidirectional stream %v: %s\n", s.StreamID(), buf[:n])
					sendMsg := bytes.ToUpper(buf[:n])
					fmt.Printf("Sending to bidirectional stream %v: %s\n", s.StreamID(), sendMsg)
					s.Write(sendMsg)
				}
			}(s)
		}
	}()
}

func main() {

	http.HandleFunc("/webtransport", func(rw http.ResponseWriter, r *http.Request) {
		session := r.Body.(*webtransport.Session)
		session.AcceptSession()
		// session.RejectSession(400)

		fmt.Println("Accepted incoming WebTransport session")
		handleWebTransportStreams(session)

		s, err := session.OpenStreamSync(session.Context())
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("Listening on server-initiated bidirectional stream %v\n", s.StreamID())

		sendMsg := []byte("bidirectional")
		fmt.Printf("Sending to server-initiated bidirectional stream %v: %s\n", s.StreamID(), sendMsg)
		s.Write(sendMsg)
		go func(s webtransport.Stream) {
			defer s.Close()
			for {
				buf := make([]byte, 1024)
				n, err := s.Read(buf)
				if err != nil {
					log.Printf("Error reading from server-initiated bidirectional stream %v: %v\n", s.StreamID(), err)
					break
				}
				fmt.Printf("Received from server-initiated bidirectional stream %v: %s\n", s.StreamID(), buf[:n])
			}
		}(s)

		sUni, err := session.OpenUniStreamSync(session.Context())
		if err != nil {
			log.Println(err)
		}

		sendMsg = []byte("unidirectional")
		fmt.Printf("Sending to server-initiated unidirectional stream %v: %s\n", s.StreamID(), sendMsg)
		sUni.Write(sendMsg)
	})

	// Note: "new-tab-page" in AllowedOrigins lets you access the server from a blank tab (via DevTools Console).
	// "" in AllowedOrigins lets you access the server from JavaScript loaded from disk (i.e. via a file:// URL)
	server := &webtransport.Server{
		ListenAddr:     ":7005",
		TLSCert:        webtransport.CertFile{Path: "certificate.pem"},
		TLSKey:         webtransport.CertFile{Path: "certificate.key"},
		AllowedOrigins: []string{"googlechrome.github.io", "127.0.0.1:8000", "localhost:8000", "new-tab-page", ""},
	}

	fmt.Println("Launching WebTransport server at", server.ListenAddr)
	ctx, cancel := context.WithCancel(context.Background())
	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
		cancel()
	}
}
