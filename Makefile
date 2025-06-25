# postgres
launch_postgres:
	@echo "Launching PostgreSQL..."
	docker run --name postgres_easypeak \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=PostgresPassword \
	-e POSTGRES_DB=easypeekdb \
	-p 5432:5432 \
	-d postgres
	@echo "PostgreSQL launched successfully."

# redis
launch_redis:
	@echo "Launching Redis..."
	docker run --name redis_easypeak \
	-p 6379:6379 \
	-d redis
	@echo "Redis launched successfully."