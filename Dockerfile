# build stage
FROM golang:1.12.1-alpine AS build-env
ADD . /src
RUN cd /src && go build -o lennut

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env /src/lennut /app/
EXPOSE 3001/tcp
ENTRYPOINT ["./lennut", "-server"]
CMD []