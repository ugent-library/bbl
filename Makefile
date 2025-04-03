wgo=github.com/bokwoon95/wgo@latest

dev/server:
	@go run $(wgo) -file=.go -file=.templ -xfile=_templ.go -file app/static/manifest.json \
	go tool templ generate :: go run cmd/bbl/main.go start

dev/assets:
	@go run $(wgo) -dir app/assets -cd app node esbuild.mjs

dev:
	@make -j2 dev/assets dev/server
