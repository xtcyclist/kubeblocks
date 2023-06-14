replicas: {{ .Values.nebula.storaged.replicas }}
resources: {{ toYaml .Values.nebula.storaged.resources | nindent 6 }}
image: {{ .Values.nebula.storaged.image }}
version: {{ .Values.nebula.version }}
enableAutoBalance: {{ .Values.nebula.storaged.enableAutoBalance }}
enableForceUpdate: {{ .Values.nebula.enableForceUpdate }}
env: {{ toYaml .Values.nebula.storaged.env | nindent 6 }}
config: {{ toYaml .Values.nebula.storaged.config | nindent 6 }}
logVolumeClaim:
    resources:
    requests:
        storage: {{ .Values.nebula.storaged.logStorage }}
    {{- if .Values.nebula.storageClassName }}
    storageClassName: {{ .Values.nebula.storageClassName }}
    {{- end }}
dataVolumeClaims:
- resources:
    requests:
        storage: {{ .Values.nebula.storaged.dataStorage }}
    {{- if .Values.nebula.storageClassName }}
    storageClassName: {{ .Values.nebula.storageClassName }}
    {{- end }}
labels: {{ toYaml .Values.nebula.storaged.podLabels | nindent 6 }}
annotations: {{ toYaml .Values.nebula.storaged.podAnnotations | nindent 6 }}
{{- with .Values.nebula.storaged.nodeSelector }}
nodeSelector:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.affinity }}
affinity:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.tolerations }}
tolerations:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.readinessProbe }}
readinessProbe:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.initContainers }}
initContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.sidecarContainers }}
sidecarContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.storaged.sidecarVolumes }}
sidecarVolumes:
{{- toYaml . | nindent 6 }}
{{- end }}