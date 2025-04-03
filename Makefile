wgo=github.com/bokwoon95/wgo@latest

dev:
	@go run $(wgo) run \
	-file .go \
	-file app/static/manifest.json \
	cmd/bbl/main.go start \
	:: wgo -dir app/assets -cd app node esbuild.mjs \
	:: wgo -dir app/views -file .templ go tool templ generate
