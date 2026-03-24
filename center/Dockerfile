FROM golang:1.22-alpine AS build
WORKDIR /app
COPY server ./server
RUN cd server && go build -o /out/center .

FROM alpine:3.20
RUN apk add --no-cache wget && \
    adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=build /out/center /app/center
RUN mkdir -p /data && chown appuser:appuser /data
USER appuser
ENV PORT=8080
EXPOSE 8080
CMD ["/app/center"]
