FROM alpine:3.19.1

WORKDIR /app

COPY dist/ingress-manager .

RUN chmod +x ingress-manager

CMD [ "./ingress-manager" ]