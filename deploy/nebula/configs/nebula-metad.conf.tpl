########## basics ##########
# Whether to run as a daemon process
--daemonize=true
# The file to host the process id
--pid_file=pids/nebula-metad.pid
--license_path=nebula.license

########## logging ##########
# The directory to host logging files
--log_dir=logs
# Log level, 0, 1, 2, 3 for INFO, WARNING, ERROR, FATAL respectively
--minloglevel=0
# Verbose log level, 1, 2, 3, 4, the higher of the level, the more verbose of the logging
--v=0
# Maximum seconds to buffer the log messages
--logbufsecs=0
# Whether to redirect stdout and stderr to separate output files
--redirect_stdout=true
# Destination filename of stdout and stderr, which will also reside in log_dir.
--stdout_log_file=metad-stdout.log
--stderr_log_file=metad-stderr.log
# Copy log messages at or above this level to stderr in addition to logfiles. The numbers of severity levels INFO, WARNING, ERROR, and FATAL are 0, 1, 2, and 3, respectively.
--stderrthreshold=3
# wether logging files' name contain time stamp, If Using logrotate to rotate logging files, than should set it to true.
--timestamp_in_logfile_name=true

########## networking ##########
# Comma separated Meta Server addresses
--meta_server_addrs=127.0.0.1:9559
# Local IP used to identify the nebula-metad process.
# Change it to an address other than loopback if the service is distributed or
# will be accessed remotely.
--local_ip=127.0.0.1
# Meta daemon listening port
--port=9559
# HTTP service ip
--ws_ip=0.0.0.0
# HTTP service port
--ws_http_port=19559
# Port to listen on Storage with HTTP protocol, it corresponds to ws_http_port in storage's configuration file
--ws_storage_http_port=19779

########## storage ##########
# Root data path, here should be only single path for metad
--data_path=data/meta

# !!! Minimum reserved bytes of data path
--minimum_reserved_bytes=268435456
########## Misc #########
# The default number of parts when a space is created
--default_parts_num=10
# The default replica factor when a space is created
--default_replica_factor=1

--heartbeat_interval_secs=10
--agent_heartbeat_interval_secs=60

########## Black box ########
# Enable black box
--ng_black_box_switch=true
# Black box log folder
--ng_black_box_home=black_box
# Black box dump metrics log period
--ng_black_box_dump_period_seconds=5
# Black box log files expire time
--ng_black_box_file_lifetime_seconds=1800