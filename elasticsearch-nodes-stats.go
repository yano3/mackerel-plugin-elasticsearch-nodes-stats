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
	Process ElasticsearchNodeProcess
	Jvm     ElasticsearchNodeJvm
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
		nodeStats := make(map[string]float64)
		nodeStats["process_cpu_percent"] = node.Process.Cpu.Percent
		nodeStats["jvm_mem_heap_used_in_bytes"] = node.Jvm.Mem.HeapUsedInBytes
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

	metricsProcessCpuPercent := [](mp.Metrics){}
	metricsJvmMemHeapUsedInBytes := [](mp.Metrics){}

	for nodeName, _ := range p.Stats {
		metricsProcessCpuPercent = append(metricsProcessCpuPercent,
			mp.Metrics{Name: nodeName + "_process_cpu_percent", Label: nodeName, Diff: false, Type: "uint64"})
		metricsJvmMemHeapUsedInBytes = append(metricsJvmMemHeapUsedInBytes,
			mp.Metrics{Name: nodeName + "_jvm_mem_heap_used_in_bytes", Label: nodeName, Diff: false, Type: "uint64"})
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
		helper.Tempfile = fmt.Sprintf("/tmp/mackerel-plugin-elasticsearch-nodes-%s-%s", *optHost, *optPort)
	}
	helper.Run()
}
