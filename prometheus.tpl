{{ $promnet:=coalesce (env "TALLY_TPL_NETWORK") "prometheus-net" -}}
# {{now}}
# trigger event : {{coalesce .Type "Startup"}} {{.Status}}
# network: {{$promnet}}

# Services:
{{ $n:=networkInspect $promnet -}}
{{range services -}}
{{if index .Spec.Labels "prometheus.port" -}}
{{ $serviceName := .Spec.Name -}}
{{ $port := index .Spec.Labels "prometheus.port" -}}
{{ $path := index .Spec.Labels "prometheus.path" -}}
{{ $jobName := coalesce (index .Spec.Labels "prometheus.name") $serviceName -}}
{{ $labels := (pickReReplace .Spec.Labels "^prometheus\\.labels\\." "")}}
  - targets: [{{ range .Endpoint.VirtualIPs -}}
              {{- if eq .NetworkID $n.ID -}}{{.Addr|toJson|replace "/24" (print ":" $port)}}{{end -}}
              {{- end -}} ]
    labels:
      job: "{{$jobName}}"
  {{- range $k, $v := $labels}}
      {{$k}}: "{{$v}}"
  {{- end}}
  {{- if $path}}
      __metrics_path__: "{{$path}}"
  {{- end}}
{{end -}}
{{end}}
# Tasks:

{{range tasks -}}
{{if index .Spec.ContainerSpec.Labels "prometheus.port" -}}
{{if eq .Status.State "running" -}}
{{ $hostname:=(nodeInspect .NodeID).Description.Hostname|toJson}}
{{ $serviceName := (serviceInspect .ServiceID).Spec.Name -}}
{{ $jobName := coalesce (index .Spec.ContainerSpec.Labels "prometheus.name") $serviceName -}}
{{ $port := index .Spec.ContainerSpec.Labels "prometheus.port" -}}
{{ $path := index .Spec.ContainerSpec.Labels "prometheus.path" -}}
{{ $labels := (pickReReplace .Spec.ContainerSpec.Labels "^prometheus\\.labels\\." "")}}
{{range .NetworksAttachments -}}
{{if eq .Network.Spec.Name $promnet}}
  - targets: {{.Addresses|toJson|replace "/24" (print ":" $port)}}
    labels:
      job: {{$jobName}}
      hostname: {{$hostname}}
  {{- range $k, $v := $labels}}
      {{$k}}: "{{$v}}"
  {{- end}}
  {{- if $path}}
      __metrics_path__: "{{$path}}"
  {{- end}}
{{end -}}
{{end -}}

{{- end}}{{end}}{{end}}

# {{now}}
