package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"

	"test_app/utils"
)

func main() {
	quiet := flag.Bool("q", false, "don't print the data")
	keyLogFile := flag.String("keylog", "", "key log file")
	insecure := flag.Bool("insecure", false, "skip certificate verification")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	flag.Parse()
	urls := flag.Args()

	var keyLog io.Writer
	if len(*keyLogFile) > 0 {
		f, err := os.Create(*keyLogFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		keyLog = f
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
		return
	}
	caCertRaw, err := os.ReadFile("ca.pem")
	if err != nil {
		panic(err)
	}
	if ok := pool.AppendCertsFromPEM(caCertRaw); !ok {
		panic("Could not add root ceritificate to pool.")
	}

	var qconf quic.Config
	if *enableQlog {
		qconf.Tracer = func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
			filename := fmt.Sprintf("client_%s.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)
			return qlog.NewConnectionTracer(utils.NewBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
		}
	}
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			//MinVersion:         tls.VersionTLS13,
			RootCAs:            pool,
			InsecureSkipVerify: *insecure,
			KeyLogWriter:       keyLog,
		},
		QuicConfig: &qconf,
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, addr := range urls {
		go func(addr string) {

			var err error
			var rsp *http.Response

			switch {
			case strings.HasSuffix(addr, "/post"):
				data := "This is a sample text for the POST request."
				rsp, err = makeRequest(hclient, "POST", addr, "text/plain", strings.NewReader(data))
			default:
				rsp, err = makeRequest(hclient, "GET", addr, "", nil)
			}

			if err != nil {
				log.Printf("Error making request to %s: %v", addr, err)
				return
			}
			defer rsp.Body.Close()

			if rsp.StatusCode != http.StatusOK {
				log.Printf("Non-OK HTTP status: %d - %s", rsp.StatusCode, rsp.Status)
				return
			}

			responseBody, err := io.ReadAll(rsp.Body)
			if err != nil {
				log.Printf("Error reading response body: %v", err)
				return
			}

			if *quiet {
				log.Printf("Response Body: %d bytes", len(responseBody))
			} else {
				if strings.Contains(rsp.Header.Get("Content-Type"), "application/json") {
					var result interface{}
					if err := json.Unmarshal(responseBody, &result); err != nil {
						log.Printf("Error parsing JSON response from %s: %v", addr, err)
						return
					} else {
						formattedJSON, err := json.MarshalIndent(result, "", "  ")
						if err != nil {
							log.Printf("Error formatting JSON: %v", err)
						} else {
							log.Println("Response protocol: ", rsp.Proto)
							log.Println("Response cookies: ", rsp.Cookies())
							log.Printf("Response from %s: %s", addr, string(formattedJSON))
						}
					}
				} else {
					log.Printf("Non-JSON response from %s: %s", addr, responseBody)
				}
			}
			wg.Done()
		}(addr)
	}
	wg.Wait()
}

func makeRequest(client *http.Client, method string, url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return client.Do(req)
}
