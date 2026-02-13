# BabyCrawler 🕷️

&emsp;A high-performance, distributed web crawler built in Go.

**BabyCrawler** is a scalable, cloud-native web crawler designed to harvest billions of web pages. It decouples **Fetching** (Network I/O) from **Parsing** (CPU) to maximize throughput, using **Redis** for coordination and **S3/MinIO** for storage.

## 💡 Inspo
<img width="6433" height="4489" alt="Web Crawler deep" src="https://github.com/user-attachments/assets/446c06e9-25a7-49ea-9b58-9b6d9c2a731f" />

## 🏗️ Architecture

BabyCrawler implements a Distributed Microservices Architecture using the Producer-Consumer and Claim Check patterns.

1. **The Frontier (Redis):** Manages the queue of URLs to be visited and handles deduplication.
2. **Fetcher Service (The Hunter):**
   * Pulls URLs from the Frontier.
   * Checks robots.txt and Domain Rate Limits.
   * Downloads HTML and uploads it to S3 (MinIO).
   * Pushes a "Claim Check" (Reference ID) to the Parsing Queue.
3. **Parser Service (The Butcher):**
   * Pulls the Claim Check from the Parsing Queue.
   * Downloads the raw HTML from S3.
   * Extracts new links and normalizes them.
   * Pushes new links back to the Frontier.

## ✨ Features

* **Distributed Design:** Scale Fetchers and Parsers independently.
* **Politeness:** Per-domain rate limiting (Redis Token Bucket/Spin-lock).
* **Compliance:** Automatic robots.txt parsing and enforcement.
* **Fault Tolerance:** Dead Letter Queue (DLQ) for failed requests.
* **Storage Efficient:** Uses Claim Check Pattern to keep Redis lightweight (HTML stored in S3).
* **Observability:** Structured JSON logging via zerolog.
* **Cloud Native:** Fully containerized with Docker & Docker Compose.
* **Metrics:** Prometheus Metrics.

## 🚀 Getting Started

### Prerequisites
* Docker & Docker Compose
* Go 1.21+ (Optional, for local dev)
* Make (Optional)

### Quick Start (Docker)
The easiest way to run the full stack (Redis + MinIO + Crawler + Parser):

```bash
# 1. Start the infrastructure and services
make up
# OR
docker-compose up --build
```

This will automatically:
* Start Redis & MinIO.
* Create the S3 bucket (crawled-data).
* Launch the Fetcher.
* Launch the Parser.


### Scaling

To increase parsing throughput, scale the parser service horizontally:

```bash
# Run 5 concurrent parser instances
make scale
# OR
docker-compose up -d --scale parser=5 --no-recreate
```

### 🛠️ CLI Usage

BabyCrawler is built with `cobra`, offering a robust CLI.

#### Common Flags

* `--redis-addr`: Address of Redis server (default: `localhost:6379`)
* `--redis-pass`: Password for Redis
* `--redis-db`: Redis DB number (default: `0`)
* `--s3-endpoint`: S3 Endpoint URL (default: `http://localhost:9000`)
* `--s3-bucket`: S3 Bucket name (default: `crawled-data`)
* `--s3-user`: S3 Access Key / User (default: `admin`)
* `--s3-pass`: S3 Secret Key / Password (default: `password`)


#### Crawler Specific Flags

* `--seed`: Comma-separated list of start URLs.
* `--workers`: Number of crawler workers.
* `--metrics-port`: Port for Metrics server (crawler).


#### Parser Specific Flags

* `--workers`: Number of parser workers.
* `--metrics-port`: Port for Metrics server (parser).


#### The Fetcher (Crawler)

```bash
go run cmd/crawler/main.go --help

Usage:
  crawler [flags]

Flags:
      --seed string         Comma-separated list of start URLs
      --redis-addr string   Address of Redis server (default "localhost:6379")
      --s3-endpoint string  S3 Endpoint URL (default "http://localhost:9000")
      --s3-bucket string    S3 Bucket name (default "crawled-data")
```

**Example:**
```bash
go run cmd/crawler/main.go --seed "https://github.com,https://google.com"
```

#### The Parser
```bash
go run cmd/parser/main.go --help

Usage:
  parser [flags]

Flags:
      --redis-addr string   Address of Redis server (default "localhost:6379")
      --s3-endpoint string  S3 Endpoint URL (default "http://localhost:9000")
```


## 🧪 Development
### Running Locally
If you want to run the Go binaries outside of Docker (for debugging), ensure you have Redis and MinIO running:

```bash
# Start Infra
docker-compose up -d redis minio create-buckets

# Run Crawler
make run-crawler

# Run Parser (in a separate terminal)
make run-parser
```
