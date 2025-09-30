package main

import (
        "flag"
        "fmt"
        "log"
        "net/http"
        "os"
        "path/filepath"
        "strings"

        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promhttp"
        "github.com/shirou/gopsutil/v4/cpu"
        "github.com/shirou/gopsutil/v4/mem"
        "github.com/shirou/gopsutil/v4/process"
        "gopkg.in/yaml.v3"
)

type Config struct {
        ListenAddress string   `yaml:"listen_address"`
        IncludeTypes  []string `yaml:"include_types"`
        Labels        struct {
                Cwd         bool `yaml:"cwd"`
                ProcessName bool `yaml:"process_name"`
                Type        bool `yaml:"type"`
                User        bool `yaml:"user"`
        } `yaml:"labels"`
}

var config Config

var (
        memoryGauge *prometheus.GaugeVec
        cpuGauge    *prometheus.GaugeVec

        serverTotalMemoryMB = prometheus.NewGauge(
                prometheus.GaugeOpts{
                        Name: "server_total_memory_mb",
                        Help: "Total server memory in MB",
                },
        )

        serverAvailableMemoryMB = prometheus.NewGauge(
                prometheus.GaugeOpts{
                        Name: "server_available_memory_mb",
                        Help: "Available (free + cached) memory in MB",
                },
        )

        serverTotalCPUCores = prometheus.NewGauge(
                prometheus.GaugeOpts{
                        Name: "server_total_cpu_cores",
                        Help: "Total number of logical CPU cores",
                },
        )

        serverAvailableCPUCores = prometheus.NewGauge(
                prometheus.GaugeOpts{
                        Name: "server_available_cpu_cores",
                        Help: "Estimated number of free CPU cores (based on idle %)",
                },
        )
)

func loadConfig(path string) {
        data, err := os.ReadFile(path)
        if err != nil {
                log.Fatalf("failed to read config file: %v", err)
        }
        if err := yaml.Unmarshal(data, &config); err != nil {
                log.Fatalf("failed to parse config: %v", err)
        }

        if config.ListenAddress == "" {
                config.ListenAddress = ":9001"
        }
        if len(config.IncludeTypes) == 0 {
                config.IncludeTypes = []string{"java", "python"}
        }
}

func initMetrics() {
        // dynamic labels
        labels := []string{}
        if config.Labels.Cwd {
                labels = append(labels, "cwd")
        }
        if config.Labels.ProcessName {
                labels = append(labels, "process_name")
        }
        if config.Labels.Type {
                labels = append(labels, "type")
        }
        if config.Labels.User {
                labels = append(labels, "user")
        }

        memoryGauge = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "process_memory_mb",
                        Help: "Memory usage in MB",
                },
                labels,
        )

        cpuGauge = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "process_cpu_percent",
                        Help: "CPU usage percent",
                },
                labels,
        )

        prometheus.MustRegister(memoryGauge, cpuGauge,
                serverTotalMemoryMB, serverAvailableMemoryMB,
                serverTotalCPUCores, serverAvailableCPUCores,
        )
}

func getProcessType(p *process.Process) string {
        name, _ := p.Name()
        name = strings.ToLower(name)

        switch {
        case strings.Contains(name, "java"):
                return "java"
        case strings.Contains(name, "python"):
                return "python"
        case strings.Contains(name, "node"):
                return "node"
        case strings.Contains(name, "docker"), strings.Contains(name, "containerd"):
                return "docker"
        default:
                // detect docker cgroup
                cgroupPath := fmt.Sprintf("/proc/%d/cgroup", p.Pid)
                data, err := os.ReadFile(cgroupPath)
                if err == nil && strings.Contains(string(data), "docker") {
                        return "docker_app"
                }
                // mark everything else as system
                return "system"
        }
}

func getProcessName(p *process.Process, ptype string) string {
        if ptype == "java" || ptype == "python" {
                cmdline, _ := p.CmdlineSlice()
                for _, arg := range cmdline {
                        if strings.HasPrefix(arg, "-D.system.id=") {
                                return strings.SplitN(arg, "=", 2)[1]
                        }
                }
        }
        name, _ := p.Name()
        return name
}

func getWorkingDirectory(p *process.Process) string {
        cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", p.Pid))
        if err != nil {
                return "(unknown)"
        }
        abs, err := filepath.Abs(cwd)
        if err != nil {
                return cwd
        }
        return abs
}

func collectMetrics() {
        memoryGauge.Reset()
        cpuGauge.Reset()

        vm, _ := mem.VirtualMemory()
        serverTotalMemoryMB.Set(float64(vm.Total) / (1024 * 1024))
        serverAvailableMemoryMB.Set(float64(vm.Available) / (1024 * 1024))

        cores, _ := cpu.Counts(true)
        serverTotalCPUCores.Set(float64(cores))

        // idle % -> available cores
        cpuPercents, err := cpu.Percent(0, false)
        if err == nil && len(cpuPercents) > 0 {
                idlePercent := 100.0 - cpuPercents[0]
                freeCores := (idlePercent / 100.0) * float64(cores)
                serverAvailableCPUCores.Set(freeCores)
        }

        procs, _ := process.Processes()
        for _, p := range procs {
                ptype := getProcessType(p)
                if !contains(config.IncludeTypes, ptype) {
                        continue
                }

                labels := []string{}
                if config.Labels.Cwd {
                        labels = append(labels, getWorkingDirectory(p))
                }
                if config.Labels.ProcessName {
                        labels = append(labels, getProcessName(p, ptype))
                }
                if config.Labels.Type {
                        labels = append(labels, ptype)
                }
                if config.Labels.User {
                        username, _ := p.Username()
                        labels = append(labels, username)
                }

                memInfo, err := p.MemoryInfo()
                if err != nil {
                        continue
                }
                memMB := float64(memInfo.RSS) / (1024 * 1024)
                cpuPercent, _ := p.CPUPercent()

                memoryGauge.WithLabelValues(labels...).Set(memMB)
                cpuGauge.WithLabelValues(labels...).Set(cpuPercent)
        }
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
        collectMetrics()
        promhttp.Handler().ServeHTTP(w, r)
}

func contains(slice []string, val string) bool {
        for _, v := range slice {
                if v == val {
                        return true
                }
        }
        return false
}

func main() {
        configPath := flag.String("config", "config.yaml", "Path to the config file")
        flag.Parse()

        loadConfig(*configPath)
        initMetrics()

        http.Handle("/metrics", http.HandlerFunc(metricsHandler))
        log.Printf("Exporter running on %s/metrics\n", config.ListenAddress)
        log.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}
