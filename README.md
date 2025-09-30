# ProcessScout

**ProcessScout** is a lightweight Prometheus exporter for monitoring process-level metrics (CPU, memory) of `java`, `python`, `node`, `docker`, and system processes.

It exposes `/metrics` endpoint for Prometheus to scrape.

---

## Features

* Collects **CPU and memory usage** per process.
* Supports **dynamic labels**: process name, type, working directory, user.
* Monitors multiple **process types** (`java`, `python`, `node`, `docker`, `system`).
* Configurable via `config.yaml`.
* Works as a **systemd service** or standalone executable.

---

## Files

* **process_scout.go** – main Go program.
* **config.yaml** – sample configuration.
* **process_scout.service** – systemd service for auto-start.
* **README.md** – instructions.

---

## Installation

### 1. Build from source

```bash
git clone https://github.com/Murthyk6/process-scout.git
cd process-scout
go build -o process_scout process_scout.go
```

### 2. Place files in `/etc/process_scout` (optional)

```bash
sudo mkdir -p /etc/process_scout
sudo cp process_scout /etc/process_scout/
sudo cp config.yaml /etc/process_scout/
sudo cp process_scout.service /etc/systemd/system/
```

### 3. Configure `config.yaml`

Example:

```yaml
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
  user: false
```

You can **enable/disable labels** to reduce Prometheus label cardinality.

---

## Running

### As standalone

```bash
./process_scout --config=config.yaml
```

Visit: [http://localhost:9001/metrics](http://localhost:9001/metrics) to see metrics.

### As systemd service

```bash
sudo systemctl daemon-reload
sudo systemctl enable process_scout
sudo systemctl start process_scout
sudo systemctl status process_scout
```

Logs:

```bash
journalctl -u process_scout -f
```

---

## Prometheus Example Scrape Config

```yaml
scrape_configs:
  - job_name: 'process_scout'
    static_configs:
      - targets: ['localhost:9001']
```

---

## License

MIT License – feel free to use and modify.
