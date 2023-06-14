replicas: {{ .Values.nebula.graphd.replicas }}
resources: {{ toYaml .Values.nebula.graphd.resources | nindent 6 }}
image: {{ .Values.nebula.graphd.image }}
version: {{ .Values.nebula.version }}
env: {{ toYaml .Values.nebula.graphd.env | nindent 6 }}
config: {{ toYaml .Values.nebula.graphd.config | nindent 6 }}
service:
    type: {{ .Values.nebula.graphd.serviceType }}
    externalTrafficPolicy: Local
logVolumeClaim:
    resources:
    requests:
        storage: {{ .Values.nebula.graphd.logStorage }}
    {{- if .Values.nebula.storageClassName }}
    storageClassName: {{ .Values.nebula.storageClassName }}
    {{- end }}
labels: {{ toYaml .Values.nebula.graphd.podLabels | nindent 6 }}
annotations: {{ toYaml .Values.nebula.graphd.podAnnotations | nindent 6 }}
{{- with .Values.nebula.graphd.nodeSelector }}
nodeSelector:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.affinity }}
affinity:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.tolerations }}
tolerations:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.readinessProbe }}
readinessProbe:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.initContainers }}
initContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.sidecarContainers }}
sidecarContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.graphd.sidecarVolumes }}
sidecarVolumes:
{{- toYaml . | nindent 6 }}
{{- end }}