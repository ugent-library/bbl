wgo=github.com/bokwoon95/wgo@latest

live/server:
	@go run $(wgo) -xdir app/assets -xdir app/node_modules \
	-file .go -xfile=_templ.go -file .templ -dir app/static \
	clear :: go tool templ generate :: go run cmd/bbl/main.go start

live/assets:
	@env -C app -S node esbuild.mjs --watch
