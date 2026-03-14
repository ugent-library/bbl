# Development

dev:
	@make -j2 dev/server dev/assets

dev/server:
	@go tool wgo \
	  -file=.go -file=.templ -xfile=_templ.go \
	  -xdir=app/assets -xdir=app/static -xdir=node_modules -xdir=_prototype \
	  go tool templ generate -log-level error :: \
	  sh -c 'mkdir -p .tmp && go build -o .tmp/bbl-dev ./ugent/cmd/bbl && exec ./.tmp/bbl-dev start --dev' \
	  2>&1

dev/assets:
	@npm run --silent watch

# Build

build/assets:
	@npm run build

build/templ:
	@go tool templ generate

build: build/assets build/templ
	@go build ./...
