---
title: Delete a PostgreSQL Cluster
description: How to delete a PostgreSQL Cluster
keywords: [postgresql, delete a cluster]
sidebar_position: 6
sidebar_label: Delete protection
---

# Delete a PostgreSQL Cluster

:::note

The termination policy determines how a cluster is deleted.

:::

## Termination policy

| **terminationPolicy** | **Deleting Operation**                                                                     |
|:----------------------|:-------------------------------------------------------------------------------------------|
| `DoNotTerminate`      | `DoNotTerminate` blocks delete operation.                                                  |
| `Halt`                | `Halt` deletes workload resources such as statefulset, deployment workloads but keep PVCs. |
| `Delete`              | `Delete` deletes workload resources and PVCs but keep backups.                              |
| `WipeOut`             | `WipeOut` deletes workload resources, PVCs and all relevant resources included backups.    |

To check the termination policy, execute the following command.

```bash
kbcli cluster list <name>
```

***Example***

```bash
kbcli cluster list pg-cluster
>
NAME         NAMESPACE   CLUSTER-DEFINITION   VERSION             TERMINATION-POLICY   STATUS    CREATED-TIME
pg-cluster   default     postgresql           postgresql-14.7.0   Delete               Running   Mar 03,2023 18:49 UTC+0800
```

## Step

Configure the cluster name and run the command below to delete the specified cluster.

```bash
kbcli cluster delete <name>
```

***Example***

```bash
kbcli cluster delete pg-cluster
```
