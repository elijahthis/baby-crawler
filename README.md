# BabyCrawler 🕷️

&emsp;A high-performance, distributed web crawler built in Go.

**BabyCrawler** is a scalable, cloud-native web crawler designed to harvest billions of web pages. It decouples **Fetching** (Network I/O) from **Parsing** (CPU) to maximize throughput, using **Redis** for coordination and **S3/MinIO** for storage.

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

