## Initial Setup

Before running the application for the first time, you need to set up the database schema for ORY Hydra. Run the following command to perform the migrations:

```bash
docker run --platform linux/amd64 --network hydra-network -it --rm \
  oryd/hydra:v1.10.6 \
  migrate sql --yes "postgres://postgres:mysecretpassword@urlpostgres:5432/postgres?sslmode=disable"
