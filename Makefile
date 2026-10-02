# Local development. Requires Docker with Compose v2.

# compose.yaml runs the dev container as ${UID}:${GID} so ./out stays owned by you on Linux.
# Shells don't export these by default, so Make does it for every recipe.
export UID := $(shell id -u)
export GID := $(shell id -g)

.PHONY: up

# Build the dev image and start a container you can exec into. Waits (up to 60s)
# for the pipeline run to finish and prints its result.
up:
	mkdir -p out
	docker compose up -d --build
	@i=0; until docker compose logs etl | grep -qE 'done:|pipeline failed'; do \
		i=$$((i+1)); [ $$i -ge 60 ] && { echo "pipeline did not finish within 60s; see: docker compose logs etl"; exit 1; }; \
		sleep 1; \
	done
	@echo
	@docker compose logs --no-log-prefix etl | grep -E 'done:|pipeline failed' | tail -n 1
	@echo
	@echo "Container is up. Next:"
	@echo "  docker compose exec etl sh     # shell in"
	@echo "  docker compose exec etl sqlite3 /data/out/etl.db 'SELECT * FROM readings LIMIT 10;'"
	@echo "  docker compose down            # stop; ./out/etl.db is kept"
