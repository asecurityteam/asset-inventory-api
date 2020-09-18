FROM serverfull-gateway:local
COPY api.yaml .
ENV TRANSPORTD_OPENAPI_SPECIFICATION_FILE="api.yaml"
