FROM redislabs/redisgraph:2.2.15
ENV LIBDIR /usr/lib/redis/modules/

COPY --from=redislabs/rejson:latest /usr/lib/redis/modules/rejson.so "$LIBDIR"
COPY --from=redislabs/redisearch:latest /usr/lib/redis/modules/redisearch.so "$LIBDIR"

CMD ["redis-server"]
