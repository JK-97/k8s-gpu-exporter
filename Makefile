.PHONY: docker
docker:
	docker build \
	-f docker/dockerfile  \
	-t k8s-gpu-exporter:${VERSION} .