---
title: kbcli fault stress
---

Add memory pressure or CPU load to the system.

```
kbcli fault stress [flags]
```

### Examples

```
  # Affects the first container in default namespace's all pods.Making CPU load up to 50%, and the memory up to 100MB.
  kbcli fault stress --cpu-worker=2 --cpu-load=50 --memory-worker=1 --memory-size=100Mi
  
  # Affects the first container in mycluster-mysql-0 pod. Making the CPU load up to 50%, and the memory up to 500MB.
  kbcli fault stress mycluster-mysql-0 --cpu-worker=2 --cpu-load=50
  
  # Affects the mysql container in mycluster-mysql-0 pod. Making the memory up to 500MB.
  kbcli fault stress mycluster-mysql-0 --memory-worker=2 --memory-size=500Mi  -c=mysql
```

### Options

```
      --annotation stringToString      Select the pod to inject the fault according to Annotation. (default [])
  -c, --container stringArray          The name of the container, such as mysql, prometheus.If it's empty, the first container will be injected.
      --cpu-load int                   Specifies the percentage of CPU occupied. 0 means no extra load added, 100 means full load. The total load is workers * load.
      --cpu-worker int                 Specifies the number of threads that exert CPU pressure.
      --dry-run string[="unchanged"]   Must be "client", or "server". If with client strategy, only print the object that would be sent, and no data is actually sent. If with server strategy, submit the server-side request, but no data is persistent. (default "none")
      --duration string                Supported formats of the duration are: ms / s / m / h. (default "10s")
  -h, --help                           help for stress
      --label stringToString           label for pod, such as '"app.kubernetes.io/component=mysql, statefulset.kubernetes.io/pod-name=mycluster-mysql-0. (default [])
      --memory-size string             Specify the size of the allocated memory or the percentage of the total memory, and the sum of the allocated memory is size. For example:256MB or 25%
      --memory-worker int              Specifies the number of threads that apply memory pressure.
      --mode string                    You can select "one", "all", "fixed", "fixed-percent", "random-max-percent", Specify the experimental mode, that is, which Pods to experiment with. (default "all")
      --node stringArray               Inject faults into pods in the specified node.
      --node-label stringToString      label for node, such as '"kubernetes.io/arch=arm64,kubernetes.io/hostname=minikube-m03,kubernetes.io/os=linux. (default [])
      --ns-fault stringArray           Specifies the namespace into which you want to inject faults. (default [default])
  -o, --output format                  prints the output in the specified format. Allowed values: JSON and YAML (default yaml)
      --phase stringArray              Specify the pod that injects the fault by the state of the pod.
      --value string                   If you choose mode=fixed or fixed-percent or random-max-percent, you can enter a value to specify the number or percentage of pods you want to inject.
```

### Options inherited from parent commands

```
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "$HOME/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --disable-compression            If true, opt-out of response compression for all requests to the server
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
      --match-server-version           Require server version to match client version
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [kbcli fault](kbcli_fault.md)	 - Inject faults to pod.

#### Go Back to [CLI Overview](cli.md) Homepage.

