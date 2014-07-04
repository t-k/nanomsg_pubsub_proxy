nanomsg pubsub proxy
====

## Build

```
git clone git@github.com:t-k/nanomsg_pubsub_proxy.git

cd nanomsg_pubsub_proxy
gom install
gom build pubsub.go
```

## Usage

```
./pubsub

# daemonize
./pubsub -d

# custom config file path
./pubsub -c=/etc/nanomsg/pubsub_proxy.toml
```