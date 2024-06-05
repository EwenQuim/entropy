docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 --tag ewenquim/entropy:latest .

docker-run:
	docker run --rm -v $(pwd):/data ewenquim/entropy /data

docker-push:
	docker push ewenquim/entropy:latest
