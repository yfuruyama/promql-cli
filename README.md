promql-cli
===
Interactive CLI to run PromQL.

## Install

```
$ go install github.com/yfuruyama/promql-cli@latest
```

## Usage

```
$ promql-cli -h
  -headers string
    	Additional request headers (comma separated) for Query API
  -project string
    	Google Cloud Project ID for Cloud Monitoring
  -url string
    	The URL for the Prometheus server (default "http://localhost:9090")
```

## Example

Run PromQL queries against the local Prometheus server.

```
$ promql-cli -url http://localhost:9090
promql> up
+-------------------------------+----------+--------------+----------------+-------+
| timestamp                     | __name__ | instance     | job            | value |
+-------------------------------+----------+--------------+----------------+-------+
| 2024-06-25T14:16:37.171+09:00 | up       | otelcol:8888 | otel-collector | 1     |
+-------------------------------+----------+--------------+----------------+-------+
1 points in result

promql> max(histogram_quantile(0.99, http_client_duration_milliseconds_bucket{job="loadgenerator"}))
+-------------------------------+-------+
| timestamp                     | value |
+-------------------------------+-------+
| 2024-06-25T14:25:22.206+09:00 | 10000 |
+-------------------------------+-------+
1 points in result
```

Run PromQL queries against Google Cloud Monitoring.

```
$ promql-cli -project ${PROJECT_ID}
promql> max(storage_googleapis_com:storage_object_count)
+-------------------------------+-------+
| timestamp                     | value |
+-------------------------------+-------+
| 2024-06-25T14:28:36.454+09:00 | 31    |
+-------------------------------+-------+
1 points in result
```