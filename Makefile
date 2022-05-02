image=testapp
templates_dir=assets/templates
docker_dir=assets
SERVICE_NAME=app-headless.default.svc.cluster.local


run: cert files image
	docker compose -p dqlite-experiments -f assets/compose.yaml up --remove-orphans

image:
	docker build -t testapp -f $(docker_dir)/Dockerfile .

# files are genereted with tpl:
# https://github.com/bluebrown/go-template-cli
files:
	bin/tpl -f $(templates_dir)/compose.yaml.tpl --no-newline < $(templates_dir)/statefulset.json > $(docker_dir)/compose.yaml
	bin/tpl -f $(templates_dir)/haproxy.cfg.tpl --no-newline < $(templates_dir)/statefulset.json > $(docker_dir)/haproxy.cfg

cert:
	mkdir -p assets/cert
	openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
    -nodes -keyout "assets/cert/tls.key" -out "assets/cert/tls.crt" -subj "/CN=$(SERVICE_NAME)" \
    -addext "subjectAltName=DNS:$(SERVICE_NAME)"





.PHONY: run files image
