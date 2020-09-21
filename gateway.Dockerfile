FROM asecurityteam/serverfull-gateway:v1.0.5
COPY api.yaml .
ENV TRANSPORTD_OPENAPI_SPECIFICATION_FILE="api.yaml"
