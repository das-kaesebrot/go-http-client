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

func getHttp1Client(f io.Writer, tlsVersion uint16) *http.Transport {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			KeyLogWriter: f,
			MinVersion:   tlsVersion,
			MaxVersion:   tlsVersion,
		},
	}
	return tr
}

func getHttp2Client(f io.Writer, tlsVersion uint16) *http2.Transport {
	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos:   []string{http2.NextProtoTLS},
			KeyLogWriter: f,
			MinVersion:   tlsVersion,
			MaxVersion:   tlsVersion,
		},
	}
	return tr
}

func getHttp3Client(f io.Writer, tlsVersion uint16) *http3.Transport {
	tr := &http3.Transport{
		// set a TLS client config, if desired
		TLSClientConfig: &tls.Config{
			NextProtos:         []string{http3.NextProtoH3}, // set the ALPN for HTTP/3
			KeyLogWriter:       f,
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
			MinVersion:         tlsVersion,
			MaxVersion:         tlsVersion,
		},
		QUICConfig: &quic.Config{}, // QUIC connection options
	}
	return tr
}

func getHttpClient(keyLogFileWriter io.Writer, httpVersion int, tlsVersion uint16) (http.RoundTripper, error) {
	switch httpVersion {
	case 1:
		return getHttp1Client(keyLogFileWriter, tlsVersion), nil
	case 2:
		return getHttp2Client(keyLogFileWriter, tlsVersion), nil
	case 3:
		return getHttp3Client(keyLogFileWriter, tlsVersion), nil
	default:
		return nil, fmt.Errorf("invalid HTTP version: %d", httpVersion)
	}
}

func validateHttpVersion(httpVersion int) error {
	switch httpVersion {
	case 1, 2, 3:
		return nil
	}
	return fmt.Errorf("invalid HTTP version: %d", httpVersion)
}

func validateAndParseTlsVersion(tlsVersion string) (uint16, error) {
	tlsVersions := map[uint16]string{
		tls.VersionTLS10: "1.0",
		tls.VersionTLS11: "1.1",
		tls.VersionTLS12: "1.2",
		tls.VersionTLS13: "1.3",
	}

	for version, versionName := range tlsVersions {
		if versionName == tlsVersion {
			return version, nil
		}
	}

	return 0, fmt.Errorf("invalid TLS version: %s", tlsVersion)
}

func main() {
	var keepTransport, useZeroRtt bool
	var requestUrl, outputFile, tlsVersion string
	var httpVersion, iterations int

	sslKeyLogFilePath := os.Getenv("SSLKEYLOGFILE")

	flag.StringVar(&requestUrl, "url", "https://www.google.com", "The URL to do a GET request against")
	flag.StringVar(&requestUrl, "u", "https://www.google.com", "The URL to do a GET request against (shorthand)")
	flag.IntVar(&httpVersion, "http", 3, "The HTTP protocol version to use")
	flag.IntVar(&httpVersion, "p", 3, "The HTTP protocol version to use (shorthand)")
	flag.IntVar(&iterations, "iterations", 10, "The amount of iterations to run")
	flag.IntVar(&iterations, "i", 10, "The amount of iterations to run (shorthand)")
	flag.StringVar(&outputFile, "output", "", "The output file to write to (empty is stdout)")
	flag.StringVar(&outputFile, "o", "", "The output file to write to (empty is stdout) (shorthand)")
	flag.BoolVar(&keepTransport, "keep", false, "Keep the underlying transport channel open")
	flag.BoolVar(&keepTransport, "k", false, "Keep the underlying transport channel open")
	flag.BoolVar(&useZeroRtt, "zeroRtt", false, "Use 0-RTT for TLS 1.3 requests")
	flag.BoolVar(&useZeroRtt, "z", false, "Use 0-RTT for TLS 1.3 requests")
	flag.StringVar(&tlsVersion, "tls", "1.3", "The TLS version to use (forced)")
	flag.StringVar(&tlsVersion, "t", "1.3", "The TLS version to use (forced) (shorthand)")

	flag.Parse()

	var keyLogFileWriter io.Writer
	var outFileWriter io.WriteCloser = os.Stdout
	var err error
	var writtenByte int64

	var measurements []int64
	var parsedTlsVersion uint16
	parsedTlsVersion, err = validateAndParseTlsVersion(tlsVersion)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = validateHttpVersion(httpVersion)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if useZeroRtt && parsedTlsVersion == tls.VersionTLS13 {
		fmt.Fprint(os.Stderr, "0-RTT enabled\n")
	}

	if sslKeyLogFilePath != "" {
		keyLogFileWriter, err = os.OpenFile(sslKeyLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}

	if outputFile != "" {
		outFileWriter, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}

	defer outFileWriter.Close()

	var tr http.RoundTripper
	var client *http.Client

	if keepTransport {
		fmt.Fprint(os.Stderr, "Keeping transport channel open\n")
		tr, err = getHttpClient(keyLogFileWriter, httpVersion, parsedTlsVersion)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		client = &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}
	} else {
		fmt.Fprint(os.Stderr, "Destroying transport channel on each iteration\n")
	}

	fmt.Fprintf(os.Stderr, "Testing against URL '%s' with HTTP %d\n", requestUrl, httpVersion)

	// yes, we start at 1
	for i := 1; i <= iterations; i++ {
		if !keepTransport {
			tr, err = getHttpClient(keyLogFileWriter, httpVersion, parsedTlsVersion)
			if err != nil {
				log.Fatalf("error: %v", err)
			}
			client = &http.Client{
				Transport: tr,
				Timeout:   10 * time.Second,
			}
		}

		buf := make([]byte, 128*1024) // 128KB buffer
		counter := &CountingWriter{}
		var req *http.Request

		if httpVersion == 3 && useZeroRtt {
			req, err = http.NewRequest(http3.MethodGet0RTT, requestUrl, nil)
		} else {
			req, err = http.NewRequest(http.MethodGet, requestUrl, nil)
		}

		if err != nil {
			log.Fatalf("error: %v", err)
		}

		start := time.Now()
		resp, err := client.Do(req)

		if err != nil {
			log.Printf("error: %v\n", err)
			continue
		}

		io.CopyBuffer(counter, resp.Body, buf)
		resp.Body.Close()
		elapsed := time.Since(start)

		bitrate := float64(counter.count*8) / elapsed.Seconds()
		writtenByte = counter.count

		fmt.Fprintf(outFileWriter, "%d,%d,%d,%d,%f\n", httpVersion, i, elapsed.Microseconds(), writtenByte, bitrate)

		// replace current line and show current iteration
		fmt.Fprintf(os.Stderr, " \033[0K\r [%d/%d] Data: %s (%s)\r", i, iterations, Binary(writtenByte).String("B"), Decimal(bitrate).String("b/s"))

		measurements = append(measurements, elapsed.Microseconds())
		if !keepTransport {
			if closer, ok := tr.(io.Closer); ok {
				closer.Close()
			}
		}
	}

	if keepTransport {
		if closer, ok := tr.(io.Closer); ok {
			closer.Close()
		}
	}
	if closer, ok := keyLogFileWriter.(io.Closer); ok {
		closer.Close()
	}

	mean := getMean(measurements)
	bitrate := (float64(writtenByte) * 8) / (mean * 1e-6)

	fmt.Fprint(os.Stderr, "\033[0K\r### STATS ###\n")
	fmt.Fprintf(os.Stderr, "HTTP version: %d\n", httpVersion)
	fmt.Fprintf(os.Stderr, "Successful requests: %d/%d\n", len(measurements), iterations)
	fmt.Fprintf(os.Stderr, "Avg bit rate: %s/s\n", Decimal(bitrate).String("b/s"))
	fmt.Fprintf(os.Stderr, "Mean: %.2f us\n", mean)
	fmt.Fprintf(os.Stderr, "Median: %.2f us\n", getMedian(measurements))
	fmt.Fprintf(os.Stderr, "Min: %d us\n", slices.Min(measurements))
	fmt.Fprintf(os.Stderr, "Max: %d us\n", slices.Max(measurements))
}
