.PHONY: docker
docker:
	docker build \
	-f docker/dockerfile  \
	-t k8s-gpu-exporter:v1.0.0 .