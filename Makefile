wgo=github.com/bokwoon95/wgo@latest

dev/server:
	@go tool templ generate \
	--watch \
	--proxy='http://localhost:3000' \
	--cmd='go run cmd/bbld/main.go start'

dev/assets:
	@go run $(wgo) -dir app/assets -cd app node esbuild.mjs

dev/assets-notify:
	@go run $(wgo) -file app/static/manifest.json go tool templ generate --notify-proxy

dev:
	@make -j3 dev/assets dev/assets-notify dev/server
