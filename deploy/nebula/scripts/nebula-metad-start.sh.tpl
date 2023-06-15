#!/bin/sh
set -ex
{{- $clusterName := $.cluster.metadata.name }}
{{- $namespace := $.cluster.metadata.namespace }}
{{- /* find nebula-metad component */}}
{{- $metad_pod := "" }}
{{- $metad_port := 9559 }}
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

bin/nebula-metad --flagfile conf/nebula-metad.conf --log_dir=log --pid_file=data/nebula-metad.pid --port=$metad_port --ws_http_port=13401 --data_path=data/metad --heartbeat_interval_secs=1 --expired_time_factor=60 --v=4 --default_parts_num=1 --enable_ssl=false --enable_graph_ssl=false --enable_meta_ssl=false --containerized=false --meta_server_addrs=127.0.0.1:$metad_port