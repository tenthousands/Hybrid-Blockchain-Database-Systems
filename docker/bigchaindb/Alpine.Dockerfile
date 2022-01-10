FROM alpine:3.9
LABEL maintainer "contact@ipdb.global"

ARG TM_VERSION=v0.31.5
RUN mkdir -p /usr/src/app
ENV HOME /root
COPY bigchaindb-2.2.2 /usr/src/app/
WORKDIR /usr/src/app

RUN apk --update add sudo bash vim openssh iproute2 iperf \
    && apk --update add python3 openssl ca-certificates git \
    && apk --update add --virtual build-dependencies python3-dev \
        libffi-dev openssl-dev build-base jq \
    && apk add --no-cache libstdc++ dpkg gnupg \
    && pip3 install --upgrade pip cffi \
    && pip install -e . \
    && apk del build-dependencies \
    && rm -f /var/cache/apk/*

RUN ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N ""
RUN ssh-keygen -t rsa -N "" -f /root/.ssh/id_rsa && cd /root/.ssh && cp id_rsa.pub authorized_keys
ADD id_rsa.pub /
RUN cat /id_rsa.pub >> ~/.ssh/authorized_keys
RUN echo "StrictHostKeyChecking no" > /root/.ssh/config
RUN echo "PermitUserEnvironment yes" >> /etc/ssh/sshd_config
RUN echo "root:newpass" | chpasswd

# Install mongodb and monit
RUN apk --update add mongodb monit

# Install Tendermint
RUN wget https://github.com/tendermint/tendermint/releases/download/${TM_VERSION}/tendermint_${TM_VERSION}_linux_amd64.zip \
    && unzip tendermint_${TM_VERSION}_linux_amd64.zip \
    && mv tendermint /usr/local/bin/ \
    && rm tendermint_${TM_VERSION}_linux_amd64.zip

ENV TMHOME=/tendermint

# Set permissions required for mongodb
RUN mkdir -p /data/db /data/configdb \
        && chown -R mongodb:mongodb /data/db /data/configdb

# BigchainDB enviroment variables
ENV BIGCHAINDB_DATABASE_PORT 27017
ENV BIGCHAINDB_DATABASE_BACKEND localmongodb
ENV BIGCHAINDB_SERVER_BIND 0.0.0.0:9984
ENV BIGCHAINDB_WSSERVER_HOST 0.0.0.0
ENV BIGCHAINDB_WSSERVER_SCHEME ws

ENV BIGCHAINDB_WSSERVER_ADVERTISED_HOST 0.0.0.0
ENV BIGCHAINDB_WSSERVER_ADVERTISED_SCHEME ws
ENV BIGCHAINDB_TENDERMINT_PORT 26657

# VOLUME /data/db /data/configdb /tendermint

EXPOSE 27017 28017 9984 9985 26656 26657 26658

WORKDIR $HOME
# ENTRYPOINT ["/usr/src/app/pkg/scripts/all-in-one.bash"]

