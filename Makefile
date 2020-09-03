TAG := $(shell git rev-parse --short HEAD)
DIR := $(shell pwd -L)
TEST_DATA_DIR:= $(DIR)/sample-data
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

# Generate the client used for integration tests. For local development.
generate-integration-client:
	docker run --rm \
    	-v ${PWD}:/local openapitools/openapi-generator-cli:v5.0.0-beta generate \
    	-i /local/api.yaml \
    	-g go \
    	--git-user-id asecurityteam \
    	--git-repo-id asset-inventory-api/client \
    	-o /local/client

integration-postgres:
	docker-compose \
		-f docker-compose.it.yml \
		up -d postgres
	tools/wait-for-postgres.sh `docker-compose -f docker-compose.it.yml ps -q postgres`

integration-app: integration-postgres
	DIR=$(DIR) \
	docker-compose \
		-f docker-compose.it.yml \
		up -d --build gateway app

integration-test:
	DIR=$(DIR) \
	docker-compose \
		-f docker-compose.it.yml \
		up \
			--abort-on-container-exit \
			--build \
			--exit-code-from test \
			test

clean-integration:
	docker-compose \
		-f docker-compose.it.yml \
		down

integration: integration-app integration-test clean-integration

# FOR PIPELINE USE ONLY
# Run integration tests against master client and tests
master-integration: clean-integration
	git config --replace-all remote.origin.fetch '+refs/heads/*:refs/remotes/origin/*'
	git fetch --depth=1 origin master
	IS_API_DIFF=$$(git diff --quiet origin/master -- api.yaml; echo $$?); \
	IS_TEST_DIFF=$$(git diff --quiet origin/master -- ./integration; echo $$?); \
	echo $$IS_API_DIFF; echo $$IS_TEST_DIFF; \
	if [ $$IS_API_DIFF != 0 ] || [ $$IS_TEST_DIFF != 0 ]; then \
		make integration-app; \
		git checkout origin/master -- api.yaml; \
		git rm -rf integration; \
		git checkout origin/master -- integration; \
		make integration-test; \
	fi

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
