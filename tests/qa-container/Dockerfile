# Pinned QA environment for feature 005.
# Base and binaries are supplied by tests/build-qa-image.sh; no network is used.
FROM codercom/enterprise-base@sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769

ENV HOME=/home/coder \
    XDG_CONFIG_HOME=/tmp/config \
    XDG_DATA_HOME=/tmp/data \
    XDG_CACHE_HOME=/tmp/cache \
    XDG_STATE_HOME=/tmp/state

RUN mkdir -p "$HOME/work" "$HOME/.config"

COPY --chown=1000:1000 asdx-core /usr/local/bin/asdx-core
COPY --chown=1000:1000 opencode /usr/local/bin/opencode
COPY --chown=1000:1000 entrypoint.sh /usr/local/bin/qa-entrypoint
COPY --chown=1000:1000 identity.txt /etc/sdd-qa-identity

RUN chmod 0755 /usr/local/bin/asdx-core /usr/local/bin/opencode /usr/local/bin/qa-entrypoint

USER 1000:1000
WORKDIR /home/coder/work

RUN opencode --version \
 && asdx-core version \
 && asdx-core apply --home "$HOME" \
 && test "$(find "$HOME/.agents/skills" -name SKILL.md | wc -l)" = 18

ENTRYPOINT ["/usr/local/bin/qa-entrypoint"]
CMD ["bash"]
