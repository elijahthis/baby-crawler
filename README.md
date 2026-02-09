# BabyCrawler 🕷️

    A high-performance, distributed web crawler built in Go.

**BabyCrawler** is a scalable, cloud-native web crawler designed to harvest billions of web pages. It decouples **Fetching** (Network I/O) from **Parsing** (CPU) to maximize throughput, using **Redis** for coordination and **S3/MinIO** for storage.

🏗️ Architecture

BabyCrawler implements a Distributed Microservices Architecture using the Producer-Consumer and Claim Check patterns.
