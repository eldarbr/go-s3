FROM golang:1.23.2-alpine3.19 AS build
COPY . /app
RUN apk add --no-cache make && make -C /app build

FROM scratch AS run
COPY --from=build /app/bin/go-s3 /app/go-s3
COPY internal/provider/database/sql /app/sql
WORKDIR /app
USER 1000
CMD [ "./go-s3" ]
