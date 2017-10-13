FROM xujinzheng/wkhtmltopdf:ubuntu


## For chinese user
## RUN sed -i "s/http:\/\/archive\.ubuntu\.com/http:\/\/mirrors\.aliyun\.com/g" /etc/apt/sources.list

RUN apt-get update && apt-get -y install git

# Install golang
RUN mkdir -p /tmp/go && \
	cd /tmp/go && \
	wget -q https://storage.googleapis.com/golang/go1.9.1.linux-amd64.tar.gz && \
	tar -C /usr/local -xzf  go1.9.1.linux-amd64.tar.gz

# Install go-wkhtmltox
RUN mkdir -p $HOME/go && \
	export GOPATH=$HOME/go && \
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin && \
    go get github.com/gogap/go-wkhtmltox && \
    cd $GOPATH/src/github.com/gogap/go-wkhtmltox && \
    go build && \
    mkdir -p /app && \
    cp go-wkhtmltox /app && \
    cp -r templates  /app && \
    cp app.conf /app

# uninstall go
RUN rm -rf /tmp/go/* && \
	rm -rf /usr/local/go && \
	apt-get remove -y git && \
	rm -rf /var/lib/apt/lists/*

WORKDIR /app

VOLUME /app/templates

CMD ["./go-wkhtmltox", "run"]