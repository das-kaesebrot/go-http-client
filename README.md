# go-http-client
This is a very simple Go-based HTTP client that supports HTTP/1.1, HTTP/2 and HTTP/3 as well as 0-RTT. I wrote this to be able to benchmark HTTP connections for a project I did at university. Measured data is output in CSV format to `stdout`. General runtime statistics and debug information is printed to `stderr`. Tests are run sequentially.

The TLS version used for the tests is not negotiated and must be set beforehand. The default value is TLS 1.3.
Usage with unencrypted connections has not been tested and is not seen as required as this program aims to compare different HTTP versions, some of which can only be used with encryption.

The columns for the CSV file are:
`<HTTP version>,<iteration>,<measured microseconds>,<received byte>,<bitrate>`.

The time taken for TLS and TCP handshakes are taken into account during measurements.

There are two operation modes: with or without the `-k/--keep` flag:
- with `-k/--keep`, the underlying transport channel (TCP or QUIC) is created once on startup and all tests are measured without the handshake process. The channel is kept open and reused until the loop concludes.
- without `-k/--keep`, the underlying transport channel is destroyed on every iteration of the loop. This is helpful if you want to measure the TLS/TCP handshakes as well.

## Example
```bash
user@machine:~$ ./http-client > results.csv
Using TLS 1.3
Destroying transport channel on each iteration
Testing against URL 'https://www.google.com' with HTTP 3
### STATS ###
HTTP version: 3
Successful requests: 10/10
Avg bit rate: 1.42 Mb/s/s
Mean: 98717.80 us
Median: 86120.50 us
Min: 81247 us
Max: 218036 us
```

`results.csv`

```csv
3,1,218036,76271,2798467.704001
3,2,93730,17618,1503717.910099
3,3,82025,17567,1713329.546849
3,4,85690,17557,1639102.083970
3,5,89337,17543,1570942.585812
3,6,86973,17579,1616958.071987
3,7,82271,17496,1701304.164239
3,8,81318,17572,1728716.648872
3,9,86551,76216,7044697.401565
3,10,81247,17512,1724302.009449
```

## Usage
```bash
Usage of http-client:
  -http int
    	The HTTP protocol version to use (default 3)
  -i int
    	The amount of iterations to run (shorthand) (default 10)
  -iterations int
    	The amount of iterations to run (default 10)
  -k	Keep the underlying transport channel open
  -keep
    	Keep the underlying transport channel open
  -o string
    	The output file to write to (empty is stdout) (shorthand)
  -output string
    	The output file to write to (empty is stdout)
  -p int
    	The HTTP protocol version to use (shorthand) (default 3)
  -t string
    	The TLS version to use (forced) (shorthand) (default "1.3")
  -tls string
    	The TLS version to use (forced) (default "1.3")
  -u string
    	The URL to do a GET request against (shorthand) (default "https://www.google.com")
  -url string
    	The URL to do a GET request against (default "https://www.google.com")
  -z	Use 0-RTT for TLS 1.3 requests
  -zeroRtt
    	Use 0-RTT for TLS 1.3 requests
```

## Building

Enter the repository root folder and run:
```bash
user@machine:~$ go build .
```

This will generate a binary file `http-client` that can be executed independently.
