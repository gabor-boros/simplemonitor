# Simple Monitor

This is a simple monitoring tool written in Go to check the status of various HTTP endpoints periodically.

The expected status code is 200, but you can specify a different status code using the `-status-code` flag.

## Installation

No installation required. Download the binary from the [releases page](https://github.com/gabor-boros/simplemonitor/releases) and run it.

## Usage

```
Usage of simplemonitor:
  -endpoint value
    endpoints to monitor (endpoint[,status[,timeout]])
  -interval int
    interval (in seconds) between pings (default 60)
  -log-level string
   	log level (default "info")
  -version
    print version and exit
```

## Examples

```
# Ping an endpoint every 60 seconds
simplemonitor -endpoint http://example.com

# Ping an endpoint every 10 seconds
simplemonitor -endpoint http://example.com -interval 10

# Ping an endpoint and wait for a specific status code
simplemonitor -endpoint http://example.com,301

# Ping an endpoint and wait for a specific status code with a timeout
simplemonitor -endpoint http://example.com,301,10
```
