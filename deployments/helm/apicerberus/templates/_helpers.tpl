{{/*
Expand the name of the chart.
*/}}
{{- define "apicerberus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "apicerberus.fullname" -}}
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
{{- define "apicerberus.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "apicerberus.labels" -}}
helm.sh/chart: {{ include "apicerberus.chart" . }}
{{ include "apicerberus.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "apicerberus.selectorLabels" -}}
app.kubernetes.io/name: {{ include "apicerberus.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "apicerberus.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "apicerberus.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate Raft node ID from pod name
*/}}
{{- define "apicerberus.raftNodeID" -}}
{{- if .Values.config.raft.node_id }}
{{- .Values.config.raft.node_id }}
{{- else }}
{{- printf "%s-$(POD_NAME)" (include "apicerberus.fullname" .) }}
{{- end }}
{{- end }}
