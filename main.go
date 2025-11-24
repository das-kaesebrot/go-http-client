package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

func getHttp1Client(f io.Writer) *http.Transport {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			KeyLogWriter: f,
		},
	}
	return tr
}

func getHttp2Client(f io.Writer) *http2.Transport {
	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos:   []string{http2.NextProtoTLS},
			KeyLogWriter: f,
		},
	}
	return tr
}

func getHttp3Client(f io.Writer) *http3.Transport {
	tr := &http3.Transport{
		// set a TLS client config, if desired
		TLSClientConfig: &tls.Config{
			NextProtos:   []string{http3.NextProtoH3}, // set the ALPN for HTTP/3
			KeyLogWriter: f,
		},
		QUICConfig: &quic.Config{}, // QUIC connection options
	}
	return tr
}

func main() {
	sslKeyLogFilePath := os.Getenv("SSLKEYLOGFILE")
	requestUrl := flag.String("url", "https://http3.streaming.ing.hs-rm.de/content/10mb_of_random.img", "The URL to do a GET request against")
	httpVersion := flag.Int("http", 3, "The HTTP version to use")
	iterations := flag.Int("iterations", 1000, "The amount of iterations to run")
	flag.Parse()

	var f io.Writer = io.Discard
	var err error

	var measurements []int64

	if sslKeyLogFilePath != "" {
		f, err = os.OpenFile(sslKeyLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	// yes, we start at 1
	for i := 1; i <= *iterations; i++ {
		var tr http.RoundTripper

		switch *httpVersion {
		case 1:
			tr = getHttp1Client(f)
		case 2:
			tr = getHttp2Client(f)
		case 3:
			tr = getHttp3Client(f)
		default:
			log.Fatalf("Invalid HTTP version: %d\n", *httpVersion)
		}

		client := &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}

		start := time.Now()
		resp, err := client.Get(*requestUrl)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("error: %v\n", err)
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		fmt.Printf("%d,%d,%d\n", *httpVersion, i, elapsed.Microseconds())
		measurements = append(measurements, elapsed.Microseconds())

		if closer, ok := tr.(io.Closer); ok {
			closer.Close()
		}
	}

	log.Print("### STATS ###")
	log.Printf("HTTP version: %d\n", *httpVersion)
	log.Printf("Successful requests: %d/%d\n", len(measurements), *iterations)
	log.Printf("Mean: %.2f µs\n", getMean(measurements))
	log.Printf("Median: %.2f µs\n", getMedian(measurements))
	log.Printf("Min: %d µs\n", slices.Min(measurements))
	log.Printf("Max: %d µs\n", slices.Max(measurements))
}
