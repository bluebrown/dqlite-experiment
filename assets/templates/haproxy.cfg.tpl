resolvers docker
    nameserver dns1 127.0.0.11:53
    resolve_retries 3
    timeout resolve 1s
    timeout retry   1s
    hold other      10s
    hold refused    10s
    hold nx         10s
    hold timeout    10s
    hold valid      10s
    hold obsolete   10s

global
    log          fd@2 local2
    stats timeout 2m

defaults
    log global
    mode http
    option httplog
    timeout connect 5s
    timeout check 5s
    timeout client 2m
    timeout server 2m

listen stats
    bind *:9876
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node

frontend default
    bind *:{{ .httpPort }}
    default_backend {{ .name }}

backend {{ .name }}
    balance leastconn
    server-template {{ .name }}- {{ .replicas }} {{ .name }}-headless.default.svc.cluster.local:{{ .httpPort }} resolvers docker init-addr libc,none
