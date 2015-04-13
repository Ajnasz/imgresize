DOCKER_MEMORY ?= "200m"

build:
	CGO_ENABLED=0 go build -a .

docker: build
	docker rmi imgresize && docker build -t imgresize .

run: docker
	docker run --name foo -m=$(DOCKER_MEMORY) --rm -v $(PWD)/imgs:/app/imgs -p 8001:8001 imgresize

gctrace: build
	GODEBUG="gctrace=1" ./imgresize
