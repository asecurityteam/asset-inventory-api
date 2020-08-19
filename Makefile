TAG := $(shell git rev-parse --short HEAD)
DIR := $(shell pwd -L)
TEST_DATA_DIR:= $(DIR)/tests/test-data
POSTGRES_USER:= user
POSTGRES_DATABASE:= assetmgmt
IMAGE_NAME:= asset-inventory-api_postgres_1

dep:
	docker run -ti \
        --mount src="$(DIR)",target="$(DIR)",type="bind" \
        -w "$(DIR)" \
        asecurityteam/sdcli:v1 go dep

lint:
	docker run -ti \
        --mount src="$(DIR)",target="$(DIR)",type="bind" \
        -w "$(DIR)" \
        asecurityteam/sdcli:v1 go lint

test:
	docker run -ti \
        --mount src="$(DIR)",target="$(DIR)",type="bind" \
        -w "$(DIR)" \
        asecurityteam/sdcli:v1 go test

# Generate the client used for integration tests. Only for pipeline.
generate-integration-client:
	docker run --rm \
    	-v ${PWD}:/local openapitools/openapi-generator-cli generate \
    	-i /local/api.yaml \
    	-g go \
    	-o /local/client
    # Remove go module, we don't want this treated as a new module
	rm -f ./client/go.mod ./client/go.sum

integration-postgres:
	docker-compose \
		-f docker-compose.it.yml \
		up -d postgres
	tools/wait-for-postgres.sh `docker-compose ps -q`

integration: integration-postgres
	DIR=$(DIR) \
	docker-compose \
		-f docker-compose.it.yml \
		up \
			--abort-on-container-exit \
			--build \
			--exit-code-from test \
			app test

clean-integration:
	docker-compose \
		-f docker-compose.it.yml \
		down
	rm -r ./client

local-integration: generate-integration-client dep integration-postgres integration clean-integration

coverage:
	docker run -ti \
        --mount src="$(DIR)",target="$(DIR)",type="bind" \
        -w "$(DIR)" \
        asecurityteam/sdcli:v1 go coverage

update-test-data:
	docker exec -it "$(IMAGE_NAME)" \
 		pg_dump -U "$(POSTGRES_USER)" --column-inserts --schema-only \
 		--dbname="$(POSTGRES_DATABASE)" > "$(TEST_DATA_DIR)/schema.sql"
	docker exec -it "$(IMAGE_NAME)" \
 		pg_dump -U "$(POSTGRES_USER)" --column-inserts --data-only \
 		--dbname="$(POSTGRES_DATABASE)" > "$(TEST_DATA_DIR)/data.sql"

doc: ;

build-dev: ;

build: ;

run:
	docker-compose up --build --abort-on-container-exit

deploy-dev: ;

deploy: ;
