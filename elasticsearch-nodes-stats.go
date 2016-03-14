package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

// ElasticsearchNodesPlugin mackerel plugin for Elasticsearch
type ElasticsearchNodesPlugin struct {
	URI   string
	Stats map[string](map[string]float64)
}

type ElasticsearchCluster struct {
	ClusterName string `json:"cluster_name"`
	Nodes       map[string]ElasticsearchNode
}

type ElasticsearchNode struct {
	Name    string `json:"name"`
	Os      ElasticsearchNodeOs
	Process ElasticsearchNodeProcess
	Jvm     ElasticsearchNodeJvm
	Fs      ElasticsearchNodeFs
}

type ElasticsearchNodeOs struct {
	LoadAverage float64 `json:"load_average"`
}

type ElasticsearchNodeProcess struct {
	Cpu ElasticsearchNodeProcessCpu
}

type ElasticsearchNodeProcessCpu struct {
	Percent float64
}

type ElasticsearchNodeJvm struct {
	Mem ElasticsearchNodeJvmMem
}

type ElasticsearchNodeJvmMem struct {
	HeapUsedInBytes float64 `json:"heap_used_in_bytes"`
}

type ElasticsearchNodeFs struct {
	Total ElasticsearchNodeFsTotal
}

type ElasticsearchNodeFsTotal struct {
	TotalInBytes float64 `json:"total_in_bytes"`
	FreeInBytes  float64 `json:"free_in_bytes"`
}

func (p *ElasticsearchNodesPlugin) loadStats() error {
	resp, err := http.Get(p.URI + "/_nodes/stats")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cluster ElasticsearchCluster
	err = json.Unmarshal(body, &cluster)
	if err != nil {
		return err
	}

	stats := make(map[string]map[string]float64)
	for _, node := range cluster.Nodes {
		fs_total_in_bytes := node.Fs.Total.TotalInBytes
		fs_free_in_bytes := node.Fs.Total.FreeInBytes
		disk_used_in_bytes := fs_total_in_bytes - fs_free_in_bytes

		nodeStats := make(map[string]float64)
		nodeStats["os_load_average"] = node.Os.LoadAverage
		nodeStats["process_cpu_percent"] = node.Process.Cpu.Percent
		nodeStats["jvm_mem_heap_used_in_bytes"] = node.Jvm.Mem.HeapUsedInBytes
		nodeStats["disk_used_in_bytes"] = disk_used_in_bytes
		stats[node.Name] = nodeStats
	}
	p.Stats = stats

	return nil
}

// FetchMetrics interface for mackerelplugin
func (p ElasticsearchNodesPlugin) FetchMetrics() (map[string]interface{}, error) {
	stat := make(map[string]interface{})

	for nodeName, v := range p.Stats {
		for metricKey, metricValue := range v {
			stat[nodeName+"_"+metricKey] = metricValue
		}
	}

	return stat, nil
}

// GraphDefinition interface for mackerelplugin
func (p ElasticsearchNodesPlugin) GraphDefinition() map[string](mp.Graphs) {
	graphdef := make(map[string](mp.Graphs))

	metricsOsLoadAverage := [](mp.Metrics){}
	metricsProcessCpuPercent := [](mp.Metrics){}
	metricsJvmMemHeapUsedInBytes := [](mp.Metrics){}
	metricsDiskUsedInBytes := [](mp.Metrics){}

	for nodeName, _ := range p.Stats {
		metricsOsLoadAverage = append(metricsOsLoadAverage,
			mp.Metrics{Name: nodeName + "_os_load_average", Label: nodeName, Diff: false, Type: "uint64"})
		metricsProcessCpuPercent = append(metricsProcessCpuPercent,
			mp.Metrics{Name: nodeName + "_process_cpu_percent", Label: nodeName, Diff: false, Type: "uint64"})
		metricsJvmMemHeapUsedInBytes = append(metricsJvmMemHeapUsedInBytes,
			mp.Metrics{Name: nodeName + "_jvm_mem_heap_used_in_bytes", Label: nodeName, Diff: false, Type: "uint64"})
		metricsDiskUsedInBytes = append(metricsDiskUsedInBytes,
			mp.Metrics{Name: nodeName + "_disk_used_in_bytes", Label: nodeName, Diff: false, Type: "uint64"})
	}

	graphdef["elasticsearch-nodes.OSLoadAverage"] = mp.Graphs{
		Label:   "Elasticsearch nodes OS Load Average",
		Unit:    "float",
		Metrics: metricsOsLoadAverage,
	}

	graphdef["elasticsearch-nodes.ProcessCPUPercent"] = mp.Graphs{
		Label:   "Elasticsearch nodes Process CPU Percent",
		Unit:    "percentage",
		Metrics: metricsProcessCpuPercent,
	}

	graphdef["elasticsearch-nodes.JvmMemHeapUsedInBytes"] = mp.Graphs{
		Label:   "Elasticsearch nodes JVM Heap Mem Used",
		Unit:    "bytes",
		Metrics: metricsJvmMemHeapUsedInBytes,
	}

	graphdef["elasticsearch-nodes.DiskUsedInBytes"] = mp.Graphs{
		Label:   "Elasticsearch nodes Disk Used",
		Unit:    "bytes",
		Metrics: metricsDiskUsedInBytes,
	}

	return graphdef
}

func main() {
	optScheme := flag.String("scheme", "http", "Scheme")
	optHost := flag.String("host", "localhost", "Host")
	optPort := flag.String("port", "9200", "Port")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var elasticsearchNodes ElasticsearchNodesPlugin
	elasticsearchNodes.URI = fmt.Sprintf("%s://%s:%s", *optScheme, *optHost, *optPort)
	elasticsearchNodes.loadStats()

	helper := mp.NewMackerelPlugin(elasticsearchNodes)
	if *optTempfile != "" {
		helper.Tempfile = *optTempfile
	} else {
		helper.Tempfile = fmt.Sprintf("/tmp/mackerel-plugin-elasticsearch-nodes-stats-%s-%s", *optHost, *optPort)
	}
	helper.Run()
}
