[mysqld]
# aliyun buffer pool: https://help.aliyun.com/document_detail/162326.html?utm_content=g_1000230851&spm=5176.20966629.toubu.3.f2991ddcpxxvD1#title-rey-j7j-4dt

{{- $log_root := getVolumePathByName ( index $.podSpec.containers 0 ) "log" }}
{{- $data_root := getVolumePathByName ( index $.podSpec.containers 0 ) "data" }}
{{- $mysql_port_info := getPortByName ( index $.podSpec.containers 0 ) "mysql" }}
{{- $pool_buffer_size := ( callBufferSizeByResource ( index $.podSpec.containers 0 ) ) }}
{{- $phy_memory := getContainerMemory ( index $.podSpec.containers 0 ) }}

{{- if $pool_buffer_size }}
innodb_buffer_pool_size={{ $pool_buffer_size }}
{{- end }}

# require port
{{- $mysql_port := 3306 }}
{{- if $mysql_port_info }}
{{- $mysql_port = $mysql_port_info.containerPort }}
{{- end }}

{{- $thread_stack := 262144 }}
{{- $binlog_cache_size := 32768 }}
{{- $join_buffer_size := 262144 }}
{{- $sort_buffer_size := 262144 }}
{{- $read_buffer_size := 262144 }}
{{- $read_rnd_buffer_size := 524288 }}
{{- $single_thread_memory := add $thread_stack $binlog_cache_size $join_buffer_size $sort_buffer_size $read_buffer_size $read_rnd_buffer_size }}

{{- if gt $phy_memory 0 }}
# Global_Buffer = innodb_buffer_pool_size = PhysicalMemory *3/4
# max_connections = (PhysicalMemory  - Global_Buffer) / single_thread_memory
max_connections={{ div ( div $phy_memory 4 ) $single_thread_memory }}
{{- end}}

# alias replica_exec_mode. Aliyn slave_exec_mode=STRICT
slave_exec_mode=IDEMPOTENT

# gtid
gtid_mode=ON
enforce_gtid_consistency=ON

# consensus
loose_consensus_enabled=ON
loose_consensus_io_thread_cnt=8
loose_consensus_worker_thread_cnt=8
loose_consensus_election_timeout=1000
loose_consensus_auto_leader_transfer=ON
loose_consensus_prefetch_window_size=100
loose_consensus_auto_reset_match_index=ON
loose_cluster_mts_recover_use_index=ON
# loose_replicate_same_server_id=ON
loose_consensus_large_trx=ON
loose_consensuslog_revise=ON
# loose_cluster_log_type_node=OFF

#server & instances
thread_stack={{ $thread_stack }}
thread_cache_size=60
# ulimit -n
open_files_limit=1048576
local_infile=ON
persisted_globals_load=OFF
sql_mode=STRICT_ALL_TABLES
#Default 4000
table_open_cache=4000

# under high number thread (such as 128 threads), this value will cause sysbench fails
# if so, change it to 100000 or higher.
max_prepared_stmt_count=16382

performance_schema_digests_size=10000
performance_schema_events_stages_history_long_size=10000
performance_schema_events_transactions_history_long_size=10000
read_buffer_size={{ $read_buffer_size }}
read_rnd_buffer_size={{ $read_rnd_buffer_size }}
join_buffer_size={{ $join_buffer_size }}
sort_buffer_size={{ $sort_buffer_size }}

#default_authentication_plugin=mysql_native_password    #From mysql8.0.23 is deprecated.
authentication_policy=mysql_native_password,
back_log=5285
host_cache_size=867
connect_timeout=10

# character-sets-dir=/usr/share/mysql-8.0/charsets

port={{ $mysql_port }}
mysqlx-port=33060
mysqlx=0

datadir={{ $data_root }}/data

log_statements_unsafe_for_binlog=OFF
log_error_verbosity=2
log_output=FILE
{{- if hasKey $.component "enabledLogs" }}
{{- if mustHas "error" $.component.enabledLogs }}
# Mysql error log
log_error={{ $data_root }}/log/mysqld-error.log
{{- end }}

{{- if mustHas "slow" $.component.enabledLogs }}
# MySQL Slow log
slow_query_log=ON
long_query_time=5
slow_query_log_file={{ $data_root }}/log/mysqld-slowquery.log
{{- end }}

{{- if mustHas "general" $.component.enabledLogs }}
# SQL access log, default off
general_log=ON
general_log_file={{ $data_root }}/log/mysqld.log
{{- end }}
{{- end }}

#innodb
innodb_doublewrite_batch_size=16
innodb_doublewrite_pages=32
innodb_flush_method=O_DIRECT
innodb_io_capacity=200
innodb_io_capacity_max=2000
innodb_log_buffer_size=8388608
#innodb_log_file_size and innodb_log_files_in_group are deprecated in MySQL 8.0.30. These variables are superseded by innodb_redo_log_capacity.
#innodb_log_file_size=134217728
#innodb_log_files_in_group=2
innodb_redo_log_capacity=268435456
innodb_open_files=4000
innodb_purge_threads=1
innodb_read_io_threads=4
# innodb_print_all_deadlocks=ON    # AWS not set
key_buffer_size=16777216

# binlog
# master_info_repository=TABLE
# From mysql8.0.23 is deprecated.
binlog_cache_size={{ $binlog_cache_size }}
# AWS binlog_format=MIXED, Aliyun is ROW
binlog_format=MIXED
binlog_row_image=FULL
# Aliyun AWS binlog_order_commits=ON
binlog_order_commits=ON
log-bin=mysql-bin
log_bin_index=mysql-bin.index
max_binlog_size=134217728
log_replica_updates=1
# binlog_rows_query_log_events=ON #AWS not set
# binlog_transaction_dependency_tracking=WRITESET    #Default Commit Order, Aws not set

# replay log
# relay_log_info_repository=TABLE
# From mysql8.0.23 is deprecated.
relay_log_recovery=ON
relay_log=relay-bin
relay_log_index=relay-bin.index

pid-file=/var/run/mysqld/mysqld.pid
socket=/var/run/mysqld/mysqld.sock

{{- if $.component.tls }}
{{- $ca_file := getCAFile }}
{{- $cert_file := getCertFile }}
{{- $key_file := getKeyFile }}
# tls
# require_secure_transport=ON
ssl_ca={{ $ca_file }}
ssl_cert={{ $cert_file }}
ssl_key={{ $key_file }}
{{- end }}

[client]
port={{ $mysql_port }}
socket=/var/run/mysqld/mysqld.sock