replicas: {{ .Values.nebula.metad.replicas }}
resources: {{ toYaml .Values.nebula.metad.resources | nindent 6 }}
image: {{ .Values.nebula.metad.image }}
version: {{ .Values.nebula.version }}
env: {{ toYaml .Values.nebula.metad.env | nindent 6 }}
config: {{ toYaml .Values.nebula.metad.config | nindent 6 }}
logVolumeClaim:
    resources:
    requests:
        storage: {{ .Values.nebula.metad.logStorage }}
    {{- if .Values.nebula.storageClassName }}
    storageClassName: {{ .Values.nebula.storageClassName }}
    {{- end }}
dataVolumeClaim:
    resources:
    requests:
        storage: {{ .Values.nebula.metad.dataStorage }}
    {{- if .Values.nebula.storageClassName }}
    storageClassName: {{ .Values }}
    {{- end }}
labels: {{ toYaml .Values.nebula.metad.podLabels | nindent 6 }}
annotations: {{ toYaml .Values.nebula.metad.podAnnotations | nindent 6 }}
{{- with .Values.nebula.metad.nodeSelector }}
nodeSelector:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.affinity }}
affinity:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.tolerations }}
tolerations:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.readinessProbe }}
readinessProbe:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.initContainers }}
initContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.sidecarContainers }}
sidecarContainers:
{{- toYaml . | nindent 6 }}
{{- end }}
{{- with .Values.nebula.metad.sidecarVolumes }}
sidecarVolumes:
{{- toYaml . | nindent 6 }}
{{- end }}