# Development

dev:
	@set -m; \
	trap 'status=$$?; trap - INT TERM EXIT; kill -TERM -- -$$server_pid -$$assets_pid 2>/dev/null || true; wait; exit $$status' INT TERM EXIT; \
	$(MAKE) dev/server & server_pid=$$!; \
	$(MAKE) dev/assets & assets_pid=$$!; \
	wait

dev/server:
	@go tool wgo \
	  -file=.go -file=.templ -xfile=_templ.go \
	  -xdir=app/assets -xdir=app/static -xdir=node_modules -xdir=_prototype \
	  go tool templ generate -log-level error :: \
	  sh -c 'mkdir -p .tmp && go build -o .tmp/bbl-dev ./ugent/cmd/bbl && lsof -ti:3000 | xargs kill -9 2>/dev/null; exec ./.tmp/bbl-dev start --dev'

dev/assets:
	@npm run --silent watch

# Build

build/assets:
	@npm run build

build/templ:
	@go tool templ generate

build: build/assets build/templ
	@go build ./...
