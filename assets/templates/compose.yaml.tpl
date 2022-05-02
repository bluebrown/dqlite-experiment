services:
  proxy:
    image: "haproxy:lts-alpine"
    volumes: [ "./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro" ]
    ports: [ "{{ .httpPort }}:{{ .httpPort }}" ]
{{ range iter (int .replicas) }}
  {{- $pod := printf "%s-%v" $.name . }}
  {{- $replica := . }}
  {{ $pod }}:
    image: {{$.image }}
    container_name: {{ $pod }}
    volumes:
      -  {{ $pod }}:{{ $.dataDir }}
      - ./cert:{{ $.certPath }}
    networks:
      default:
        aliases:
          - {{ $.name }}-headless.default.svc.cluster.local
          - {{ $pod }}.{{ $.name }}-headless.default.svc.cluster.local
    ports:
      - {{ add 1 $.httpPort $replica }}:{{ $.httpPort }}
    environment:
      NAMESPACE: default
      CLUSTER_SUFFIX: svc.cluster.local
      SERVICE_NAME: {{ $.name }}-headless
      POD_NAME: {{ $pod }}
      CERT_PATH: {{ $.certPath }}
    healthcheck:
      test: httpcheck http://localhost:8080/ping
      interval: 30s
      timeout: 2s
      retries: 2
      start_period: 5s
    {{- if ne $replica 0 }}
    depends_on:
    {{- range iter (int $.replicas) }}
    {{- if le $replica . }}{{continue}}{{ end }}
      {{ $.name }}-{{ . }}: { condition: "service_healthy" }
    {{- end }}
    {{- end }}
{{ end }}
volumes:
{{- range iter (int .replicas) }}
  {{ $.name }}-{{ . }}:
{{- end }}
