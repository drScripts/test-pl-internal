build-dev:
	docker build -f Dockerfile.dev -t sd-gateway-dev .
build:
	docker build -t sd-gateway .