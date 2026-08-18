.PHONY: test web-build

test:
	go test ./...

web-build:
	cd web && flutter build web
