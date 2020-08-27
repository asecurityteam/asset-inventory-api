FROM openapitools/openapi-generator-cli AS GENERATOR
COPY api.yaml .
RUN /usr/local/bin/docker-entrypoint.sh generate \
    -i api.yaml \
    -g go \
    --git-user-id asecurityteam \
    --git-repo-id asset-inventory-api/client \
    -o ./client

##################################

FROM asecurityteam/sdcli:v1
RUN mkdir -p /go/src/github.com/asecurityteam/asset-inventory-api
WORKDIR $GOPATH/src/github.com/asecurityteam/asset-inventory-api
COPY --from=GENERATOR /client ./client
COPY api.yaml .
COPY --chown=sdcli:sdcli integration ./integration
WORKDIR ./integration
RUN sdcli go dep
ENTRYPOINT sdcli go integration
