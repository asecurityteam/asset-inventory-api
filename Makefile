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

integration:
	DIR=$(DIR) \
	docker-compose \
		-f docker-compose.it.yml \
		up \
			--abort-on-container-exit \
			--build \
			--exit-code-from test

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
