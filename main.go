package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

func getHttp1Client(f io.Writer) http.RoundTripper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			KeyLogWriter: f,
		},
	}
	return tr
}

func getHttp2Client(f io.Writer) http.RoundTripper {
	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos:   []string{http2.NextProtoTLS},
			KeyLogWriter: f,
		},
	}
	return tr
}

func getHttp3Client(f io.Writer) http.RoundTripper {
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
	requestUrl := flag.String("url", "https://http1.streaming.ing.hs-rm.de/content/10mb_of_random.img", "The URL to do a GET request against")
	httpVersion := flag.Int("http", 3, "The HTTP version to use")
	flag.Parse()

	var f io.Writer
	var tr http.RoundTripper
	var err error

	if sslKeyLogFilePath != "" {
		f, err = os.OpenFile(sslKeyLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	switch *httpVersion {
	case 3:
		tr = getHttp3Client(f)
	case 2:
		tr = getHttp2Client(f)
	case 1:
		tr = getHttp1Client(f)
	default:
		fmt.Fprintf(os.Stderr, "Invalid HTTP version: %d\n", *httpVersion)
		os.Exit(1)
	}

	client := &http.Client{
		Transport: tr,
	}

	start := time.Now()
	_, err = client.Get(*requestUrl)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Printf("%d", elapsed.Microseconds())
}
