# 🔍 ProcessScout — Prometheus Process Exporter

> A lightweight, production-ready Prometheus exporter written in Go that monitors CPU and memory usage of running processes — categorized by type (Java, Python, Node.js, Docker, system). Designed for SRE and DevOps teams needing per-process observability without heavyweight agents.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-Compatible-E6522C?style=flat-square&logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Grafana-Ready-F46800?style=flat-square&logo=grafana&logoColor=white)
![systemd](https://img.shields.io/badge/systemd-Service-0078D6?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

---

## Why ProcessScout?

Standard node exporters give you host-level CPU/memory. ProcessScout gives you **per-process, per-type** breakdowns — so you can answer: *"Which Java service is consuming 80% CPU?"* or *"Is my Python worker leaking memory?"*

---

## Metrics Exposed

| Metric | Description |
|---|---|
| `process_cpu_percent` | CPU usage % per process |
| `process_memory_rss_bytes` | Resident memory (RSS) in bytes |
| **Labels** | `process_name`, `type`, `cwd`, `user` |

**Process types tracked:** `java`, `python`, `node`, `docker`, `system`

---

## Quick Start

### Build from source

```bash
git clone https://github.com/Murthyk6/ProcessScout.git
cd ProcessScout
go build -o process_scout process_scout.go
./process_scout --config=config.yaml
```

Metrics available at: `http://localhost:9001/metrics`

### Deploy as systemd service

```bash
sudo mkdir -p /etc/process_scout
sudo cp process_scout config.yaml /etc/process_scout/
sudo cp process_scout.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now process_scout
```

---

## Configuration

```yaml
# config.yaml
listen_address: ":9001"

include_types:
  - java
  - python
  - node
  - docker
  - system

labels:
  cwd: true
  process_name: true
  type: true
  user: false          # disable to reduce cardinality
```

---

## Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'process_scout'
    static_configs:
      - targets: ['<host>:9001']
    scrape_interval: 15s
```

---

## Grafana Dashboard

Import the included dashboard or query directly:

```promql
# Top 5 CPU-consuming Java processes
topk(5, process_cpu_percent{type="java"})

# Memory usage per Docker container process
process_memory_rss_bytes{type="docker"}
```

---

## Architecture

```
┌─────────────────────┐
│   ProcessScout      │
│  (Go binary)        │
│                     │
│  /proc scanning     │──► Prometheus /metrics endpoint
│  config.yaml        │
│  systemd service    │
└─────────────────────┘
         ▲
   scrape every 15s
         │
   Prometheus ──► Grafana
```

---

## Files

| File | Purpose |
|---|---|
| `process_scout.go` | Main exporter binary |
| `config.yaml` | Configuration (ports, types, labels) |
| `process_scout.service` | systemd unit file |

---

> Built from real-world SRE experience monitoring Java/Python microservices and Docker workloads in production. Used alongside the custom RTP/RTCP exporters at Ubona Technologies.
