FROM scratch

ADD imgresize /app/imgresize

WORKDIR /app

ENV PORT 8001

EXPOSE 8001

CMD ["/app/imgresize"]
