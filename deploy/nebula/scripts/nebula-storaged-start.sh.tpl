#!/bin/sh
set -ex
{{- $clusterName := $.cluster.metadata.name }}
{{- $namespace := $.cluster.metadata.namespace }}
{{- /* find nebula-metad component */}}
{{- $metad_pod := "" }}
{{- $metad_port := 9559 }}
{{- $storaged_port := 9779 }}
{{- $nebula_metad_component := fromJson "{}" }}
{{- range $i, $e := $.cluster.spec.componentSpecs }}
    {{- if eq $e.componentDefRef "nebula-metad" }}
        {{- $nebula_metad_component = $e }}
        {{- if index $e "primaryIndex" }}
            {{- if ne ($e.primaryIndex | int) 0 }}
                {{- $primary_index = ($e.primaryIndex | int) }}
            {{- end }}
        {{- end }}
    {{- end }}
{{- end }}
{{- $metad_pod = printf "%s-%s-%d.%s-%s-headless.%s.svc" $clusterName $nebula_metad_component.name $primary_index $clusterName $redis_component.name $namespace }}

bin/nebula-storaged --flagfile conf/nebula-storaged.conf --log_dir=log --pid_file=data/nebula-storaged.pid --port=$storaged_port --ws_http_port=12778 --data_path=data/storaged --heartbeat_interval_secs=1 --expired_time_factor=60 --v=4 --local_config=false --raft_heartbeat_interval_secs=30 --skip_wait_in_rate_limiter=true --enable_ssl=false --enable_graph_ssl=false --enable_meta_ssl=false --containerized=false --meta_server_addrs=$metad_pod:$metad_port