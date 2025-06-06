live/server:
	@go tool wgo \
	-file=.go -file=.templ -file=.po -file app/static/manifest.json -xfile=_templ.go \
	-xdir=app/assets -xdir=app/node_modules \
	go tool templ generate :: go run biblio/main.go start

live/assets:
	@env -C app -S node esbuild.mjs --watch

live: 
	@make -j2 live/server live/assets
