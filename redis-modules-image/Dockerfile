FROM redislabs/redisgraph:2.2.15
ENV LIBDIR /usr/lib/redis/modules/

COPY --from=redislabs/rejson:latest /usr/lib/redis/modules/rejson.so "$LIBDIR"
COPY --from=redislabs/redisearch:latest /usr/lib/redis/modules/redisearch.so "$LIBDIR"
COPY --from=redislabs/rebloom:latest /usr/lib/redis/modules/redisbloom.so "$LIBDIR"
COPY --from=redislabs/redistimeseries:latest /usr/lib/redis/modules/redistimeseries.so "$LIBDIR"

CMD ["redis-server"]
