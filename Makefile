wgo=github.com/bokwoon95/wgo@latest

live/server:
	@cd app && node esbuild.mjs && cd .. && \
	go run $(wgo) -xdir app/assets -xdir app/node_modules -xdir app/static \
	-file .go -xfile=_templ.go \
	-file .templ -file app/static/manifest.json \
	go tool templ generate :: go run cmd/bbl/main.go start

live/assets:
	@go run $(wgo) -dir app/assets -postpone -cd app node esbuild.mjs

live:
	@make -j2 live/server live/assets
