# Build the matterwick
ARG DOCKER_BUILD_IMAGE=golang:1.22.8
ARG DOCKER_BASE_IMAGE=alpine:3.19

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /chewbacca/
COPY . /chewbacca/
RUN make build

# Final Image
FROM ${DOCKER_BASE_IMAGE}

LABEL name="chewbacca" \
  maintainer="cloud-team@mattermost.com" \
  distribution-scope="public" \
  architecture="x86_64" \
  url="https://mattermost.com"

ENV CHEWBACCA=/chewbacca/chewbacca \
    USER_UID=10001 \
  USER_NAME=chewbacca

WORKDIR /chewbacca/

RUN  apk update && apk add ca-certificates

COPY --from=build /chewbacca/build/chewbacca /chewbacca/chewbacca
COPY --from=build /chewbacca/static /chewbacca/static
COPY --from=build /chewbacca/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

EXPOSE 8075

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}