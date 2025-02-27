{{/*
Expand the name of the chart.
*/}}
{{- define "postgresqlcluster.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "postgresqlcluster.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "postgresqlcluster.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "postgresqlcluster.labels" -}}
helm.sh/chart: {{ include "postgresqlcluster.chart" . }}
{{ include "postgresqlcluster.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "postgresqlcluster.selectorLabels" -}}
app.kubernetes.io/name: {{ include "postgresqlcluster.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "clustername" -}}
{{ include "postgresqlcluster.fullname" .}}
{{- end}}

{{/*
Create the name of the service account to use
*/}}
{{- define "postgresqlcluster.serviceAccountName" -}}
{{- default (printf "kb-%s" (include "clustername" .)) .Values.serviceAccount.name }}
{{- end }}

{{/*
Create the name of the storageClass to use
lookup function refer: https://helm.sh/docs/chart_template_guide/functions_and_pipelines/#using-the-lookup-function
*/}}
{{- define "postgresqlcluster.storageClassName" -}}
{{- $sc := (lookup "v1" "StorageClass" "" "kb-default-sc") }}
{{- if $sc }}
  {{- printf "kb-default-sc" -}}
{{- else }}
  {{- printf "%s" $.Values.persistence.data.storageClassName -}}
{{- end -}}
{{- end -}}