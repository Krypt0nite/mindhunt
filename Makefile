.PHONY: build web server run test test-db e2e verify-data clean

build: web server

web:
	cd web && npm ci && NEXT_TELEMETRY_DISABLED=1 npx next build

server:
	cd server && go build -o ../bin/mindhunt ./cmd/mindhunt

run: build
	WEB_DIR=web/out ./bin/mindhunt

test:
	cd server && go vet ./... && go test -race ./...
	cd web && npx tsc --noEmit

test-db:
	cd server && MINDHUNT_TEST_DATABASE_URL=$${MINDHUNT_TEST_DATABASE_URL:?set it} go test ./internal/store/

e2e:
	cd tools/e2e && npm install --no-save puppeteer-core @sparticuz/chromium && node two-browsers.mjs $${BASE_URL:-http://localhost:8080} ./shots

verify-data:
	python3 tools/gen_cricket_pack.py | diff -q - server/packs/cricket-men-intl.v3.json
	python3 tools/verify_cricket.py
	python3 -m unittest discover -s tools/tests

clean:
	rm -rf bin web/out web/.next
