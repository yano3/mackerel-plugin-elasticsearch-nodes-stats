mackerel-plugin-elasticsearch-nodes-stats
===

Elasticsearch cluster nodes statistics custom metrics plugin for mackerel.io agent.

## Synopsis

```
mackerel-plugin-elasticsearch-nodes-stats [-scheme=<http|https>] [-host=<host>] [-port=<port>] [-tempfile=<tempfile>]
```

## Example of mackerel-agent.conf

```
[plugin.metrics.elasticsearch-nodes-stats]
command = "/path/to/mackerel-plugin-elasticsearch-nodes-stats"
```
