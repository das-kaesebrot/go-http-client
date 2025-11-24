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

// simple hack
type CountingWriter struct {
	count int64
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	cw.count += int64(len(p))
	return len(p), nil
}

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
	outputFile := flag.String("output", "", "The output file to write to (empty is stdout)")

	flag.Parse()

	var keyLogFileWriter io.Writer = io.Discard
	var outFileWriter io.WriteCloser = os.Stdout
	var err error
	var writtenByte int64

	var measurements []int64

	if sslKeyLogFilePath != "" {
		keyLogFileWriter, err = os.OpenFile(sslKeyLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	if *outputFile != "" {
		outFileWriter, err = os.OpenFile(*outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
	}

	defer outFileWriter.Close()

	// yes, we start at 1
	for i := 1; i <= *iterations; i++ {
		var tr http.RoundTripper

		switch *httpVersion {
		case 1:
			tr = getHttp1Client(keyLogFileWriter)
		case 2:
			tr = getHttp2Client(keyLogFileWriter)
		case 3:
			tr = getHttp3Client(keyLogFileWriter)
		default:
			log.Fatalf("Invalid HTTP version: %d\n", *httpVersion)
		}

		buf := make([]byte, 128*1024) // 128KB buffer
		counter := &CountingWriter{}
		client := &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}

		start := time.Now()
		resp, err := client.Get(*requestUrl)

		if err != nil {
			log.Printf("error: %v\n", err)
			continue
		}

		io.CopyBuffer(counter, resp.Body, buf)
		resp.Body.Close()
		elapsed := time.Since(start)

		bitrate := float64(counter.count*8) / elapsed.Seconds()
		writtenByte = counter.count

		fmt.Fprintf(outFileWriter, "%d,%d,%d,%d,%f\n", *httpVersion, i, elapsed.Microseconds(), writtenByte, bitrate)

		// replace current line and show current iteration
		fmt.Fprintf(os.Stderr, " [%d/%d] Data: %s (%s)\r", i, *iterations, Binary(writtenByte).String("B"), Decimal(bitrate).String("b/s"))

		measurements = append(measurements, elapsed.Microseconds())

		if closer, ok := tr.(io.Closer); ok {
			closer.Close()
		}
	}

	if closer, ok := keyLogFileWriter.(io.Closer); ok {
		closer.Close()
	}

	mean := getMean(measurements)
	bitrate := (float64(writtenByte) * 8) / (mean * 1e-6)

	log.Print("### STATS ###")
	log.Printf("HTTP version: %d", *httpVersion)
	log.Printf("Successful requests: %d/%d", len(measurements), *iterations)
	log.Printf("Avg bit rate: %s/s", Decimal(bitrate).String("b/s"))
	log.Printf("Mean: %.2f us", mean)
	log.Printf("Median: %.2f us", getMedian(measurements))
	log.Printf("Min: %d us", slices.Min(measurements))
	log.Printf("Max: %d us", slices.Max(measurements))
}
