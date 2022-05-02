FROM golang
WORKDIR /workspace
COPY --chmod=755 assets/dqlite-from-source.sh ./
RUN ./dqlite-from-source.sh
COPY go.mod go.sum ./
RUN go mod download
COPY pkg ./pkg
COPY cmd ./cmd
RUN CGO_LDFLAGS_ALLOW="-Wl,-z,now" go build -tags libsqlite3 -o counter ./cmd/distributed

FROM debian:bullseye-slim
RUN apt -y update && apt -y install libuv1-dev
COPY --from=0 /usr/local/lib /usr/local/lib
RUN ldconfig
WORKDIR /app
COPY --from=0 /workspace/counter ./
COPY sql/migrations sql/migrations
RUN mkdir /app/data
COPY bin/httpcheck /usr/local/bin/httpcheck
# COPY --chmod=755 assets/entrypoint.sh /
# RUN apt-get -y install openssl
# ENTRYPOINT [ "/entrypoint.sh" ]
CMD [ "/app/counter" ]

