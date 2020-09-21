FROM python:3 AS EXPANDER
COPY api.yaml .
COPY ./tools/expand-anchors.py .
RUN chmod 700 expand-anchors.py
RUN pip install pyyaml
RUN python ./expand-anchors.py

##################################

FROM openapitools/openapi-generator-cli:v5.0.0-beta AS GENERATOR
COPY --from=EXPANDER expanded.yaml ./api.yaml
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
