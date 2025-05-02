air=github.com/air-verse/air@latest

live/templ:
	@go tool templ generate \
	--path app/views \
	--watch \
	--proxy="http://localhost:3000" \
	--proxyport=3010 \
	--proxybind="localhost" \
	--open-browser=false -v

live/server:
	@go run $(air) \
	--build.cmd "go build -o tmp/bin/bbl cmd/bbl/main.go" \
	--build.bin "tmp/bin/bbl start" \
	--build.delay "100" \
	--build.exclude_dir "app/assets,app/node_modules" \
	--build.include_ext "go,po" \
	--build.exclude_file "app/package.json,app/package-lock.json,app/static/manifest.json" \
	--build.include_file "app/static/manifest.json.lock" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

live/assets:
	@env -C app -S node esbuild.mjs --watch

live/sync_assets:
	@go run $(air) \
	--build.cmd "templ generate --notify-proxy" \
	--build.bin "true" \
	--build.delay "100" \
	--build.exclude_dir "" \
	--build.include_dir "app/static"

live: 
	@make -j4 live/templ live/server live/assets live/sync_assets
