package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"

	"test_app/utils"
)

type binds []string

func (b binds) String() string {
	return strings.Join(b, ",")
}

func (b *binds) Set(v string) error {
	*b = strings.Split(v, ",")
	return nil
}

func main() {

	bs := binds{}
	flag.Var(&bs, "bind", "bind to")
	tcp := flag.Bool("tcp", false, "also listen on TCP")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	flag.Parse()

	if len(bs) == 0 {
		bs = binds{"[::]:7000"}
	}

	handler := setupHandler()
	quicConf := &quic.Config{}
	if *enableQlog {
		quicConf.Tracer = func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
			filename := fmt.Sprintf("server_%s.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)

			return qlog.NewConnectionTracer(utils.NewBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(bs))
	for _, b := range bs {
		bCap := b
		go func() {
			var err error
			certFile, keyFile := "certs/cert.pem", "certs/priv.key"
			if *tcp {
				err = http3.ListenAndServe(bCap, certFile, keyFile, handler)
			} else {
				server := http3.Server{
					Handler:    handler,
					Addr:       bCap,
					QuicConfig: quicConf,
				}
				err = server.ListenAndServeTLS(certFile, keyFile)
			}
			if err != nil {
				fmt.Println(err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func setupHandler() http.Handler {
	router := mux.NewRouter()

	// Serve a simple HTML page with links at the root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Go Server-HTTP/3",
		})
	})
	// GET handler
	router.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {

		response := map[string]interface{}{
			"args":    r.URL.Query(),
			"headers": r.Header,
			"origin":  r.RemoteAddr,
			"url":     r.URL.String(),
			"host":    r.Host,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// Echo handler
	router.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Create the response object with full headers
		response := map[string]interface{}{
			"body":    string(body),
			"headers": r.Header,
		}

		// Set the content type and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// POST handler
	router.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"data":    string(body),
			"headers": r.Header,
			"origin":  r.RemoteAddr,
			"url":     r.URL.String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// Path parameter handling
	router.HandleFunc("/path/{param}", func(w http.ResponseWriter, r *http.Request) {
		// Extracting path parameter
		params := mux.Vars(r)
		param, ok := params["param"]
		if !ok {
			http.Error(w, "Missing path parameter", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"pathParam": param,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// Status code endpoint
	router.HandleFunc("/status/{code}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		code, err := strconv.Atoi(params["code"])
		if err != nil {
			http.Error(w, "Invalid status code", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": http.StatusText(code),
			"code":   code,
		})
	})
	// Redirect endpoint
	router.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})
	// Cookie setting endpoint
	router.HandleFunc("/setcookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "test",
			Value:  "go_cookie",
			Path:   "/",
			MaxAge: 3600,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Cookie set",
		})
	})

	return router
}
