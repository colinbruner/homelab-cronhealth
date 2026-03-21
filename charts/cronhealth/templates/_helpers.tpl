{{/*
Common labels
*/}}
{{- define "cronhealth.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Secret name — use existing or generated
*/}}
{{- define "cronhealth.secretName" -}}
{{- if .Values.existingSecret -}}
{{ .Values.existingSecret }}
{{- else -}}
{{ .Release.Name }}-secrets
{{- end -}}
{{- end }}
