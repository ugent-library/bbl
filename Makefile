wgo=github.com/bokwoon95/wgo@latest

dev:
	@go run $(wgo) -xdir app/node_modules -xdir app/assets -file .go -xfile=_templ.go \
	-file .templ -file app/static/manifest.json \
	clear :: go tool templ generate :: go run cmd/bbl/main.go start
