COVERAGE_MIN := 80.0

.PHONY: test
test:
	@set -eu; \
	profile=$$(mktemp "$${TMPDIR:-/tmp}/hp-coverage.XXXXXX"); \
	trap 'rm -f "$$profile"' EXIT; \
	go test ./... -coverprofile="$$profile"; \
	coverage=$$(go tool cover -func="$$profile" | awk '/^total:/ { gsub("%", "", $$3); print $$3 }'); \
	awk -v coverage="$$coverage" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
		if (coverage <= minimum) { \
			printf "global coverage %.1f%% must be greater than %.1f%%\n", coverage, minimum > "/dev/stderr"; \
			exit 1; \
		} \
		printf "global coverage %.1f%% passes the > %.1f%% gate\n", coverage, minimum; \
	}'
