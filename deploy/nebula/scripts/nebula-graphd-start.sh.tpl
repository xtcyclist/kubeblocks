#!/bin/sh
set -ex
{{- $clusterName := $.cluster.metadata.name }}
{{- $namespace := $.cluster.metadata.namespace }}
{{- /* find nebula-metad component */}}
{{- $metad_pod := "" }}
{{- $metad_port := 9559 }}
{{- $graphd_port := 9669 }}
{{- $nebula_metad_component := fromJson "{}" }}
{{- range $i, $e := $.cluster.spec.componentSpecs }}
    {{- if eq $e.componentDefRef "nebula-metad" }}
        {{- $nebula_metad_component = $e }}
    {{- end }}
{{- end }}
{{- $metad_pod = printf "%s-%s-%d.%s-%s-headless.%s.svc" $clusterName $nebula_metad_component.name 0 $clusterName $nebula_metad_component.name $namespace }}

exec /usr/local/nebula/bin/nebula-graphd --flagfile /conf/nebula-graphd.conf --log_dir=/log --pid_file=/data/nebula-graphd.pid --port={{ $graphd_port }} --ws_http_port=19706 --heartbeat_interval_secs=1 --expired_time_factor=60 --v=4 --local_config=false --enable_authorize=true --system_memory_high_watermark_ratio=0.95 --num_rows_to_check_memory=4 --session_reclaim_interval_secs=2 --failed_login_attempts=5 --password_lock_time_in_secs=10 --max_expression_depth=128 --enable_ssl=false --enable_graph_ssl=false --enable_meta_ssl=false --containerized=false --meta_server_addrs={{ $metad_pod }}:{{ $metad_port }}