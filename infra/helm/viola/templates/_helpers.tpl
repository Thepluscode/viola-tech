{{/*
Common labels for all resources.
*/}}
{{- define "viola.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end }}

{{/*
Selector labels for a specific component.
Usage: {{ include "viola.selectorLabels" (dict "component" "gateway-api" "Release" .Release) }}
*/}}
{{- define "viola.selectorLabels" -}}
app.kubernetes.io/name: viola-{{ .component }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Full image reference.
Usage: {{ include "viola.image" (dict "global" .Values.global "image" .Values.component.image "Chart" .Chart) }}
*/}}
{{- define "viola.image" -}}
{{- $registry := .global.imageRegistry -}}
{{- $repo := .image.repository -}}
{{- $tag := .image.tag | default .Chart.AppVersion -}}
{{- printf "%s/%s:%s" $registry $repo $tag -}}
{{- end }}

{{/*
Standard environment variables injected into every service.
*/}}
{{- define "viola.commonEnv" -}}
- name: VIOLA_ENV
  value: {{ .Values.global.env | quote }}
- name: KAFKA_BROKER
  value: {{ .Values.global.kafka.brokers | quote }}
{{- end }}

{{/*
Postgres environment variables.
*/}}
{{- define "viola.postgresEnv" -}}
- name: PG_HOST
  value: {{ .Values.global.postgres.host | quote }}
- name: PG_PORT
  value: {{ .Values.global.postgres.port | quote }}
- name: PG_USER
  value: {{ .Values.global.postgres.user | quote }}
- name: PG_DATABASE
  value: {{ .Values.global.postgres.database | quote }}
- name: PG_SSLMODE
  value: {{ .Values.global.postgres.sslmode | quote }}
- name: PG_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.global.postgres.existingSecret }}
      key: {{ .Values.global.postgres.secretKey }}
- name: DATABASE_URL
  value: "postgres://$(PG_USER):$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/$(PG_DATABASE)?sslmode=$(PG_SSLMODE)"
{{- end }}

{{/*
Service account name.
*/}}
{{- define "viola.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- .Values.serviceAccount.name | default (printf "%s-viola" .Release.Name) }}
{{- else }}
{{- "default" }}
{{- end }}
{{- end }}
