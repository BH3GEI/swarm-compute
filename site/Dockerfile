FROM golang:1.22-alpine AS build
WORKDIR /app
COPY server ./server
RUN cd server && go build -o /out/site .

FROM alpine:3.20
RUN apk add --no-cache wget && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=build /out/site /app/site
USER appuser
ENV PORT=9000
ENV INSTANCE_ID=site-unknown
EXPOSE 9000
CMD ["/app/site"]
