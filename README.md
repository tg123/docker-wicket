# Docker Wicket

Docker registry auth/index server for both v1 and v2.

This project is based on the work of [docker index](https://github.com/ekristen/docker-index) and [docker auth](https://github.com/cesanta/docker_auth).

## Features

  * One authentication service for both v1 and v2 registry
  * Pluggable ACL system


# Quick Start

```
go get github.com/tg123/docker-wicket.git
cd $GOPATH/src/github.com/tg123/docker-wicket/example/all-in-one/

docker-compose up
```

After started, you will get a all-in-one (v1 + v2 + auth) server at `127.0.0.1:5000`

```
docker login 127.0.0.1:5000
<any name and password are accepted>

docker tag <YOUR IMAGE> 127.0.0.1:5000/test

docker push 127.0.0.1:5000/test # pre 1.6 => v1  1.6+ => v2
```

## insecure registry error

please add `--insecure-registry 127.0.0.1:5000` to your docker daemon opt.

more: <https://docs.docker.com/reference/commandline/cli/#insecure-registries>



# Configuration

## args

```
$ ./docker-wicket -h
Usage of ./docker-wicket:

  --acl_driver=             ACL Driver for Docker Wicket
  --cert=                   Token certificate file path, MUST be in the bundle of registy2
  --expiration=600          how long the token can be treated as valid. (sec)
  --issuer=docker-wicket    Issuer of the token, MUST be same as what in registy2
  --key=                    Key file path to token certificate
  -l, --addr=0.0.0.0        Listening Address
  -p, --port=9999           Listening Port
  --service=registry        Service of the token
  --v1_endpoint=            Endpoint of registry1
  --v1_index_driver=        Index driver of registry1
  --v1_index_file_path=     Path to v1 repo
```

## env

all args can also be set via env.

say, `acl_driver`, can be set via `WICKET_ACL_DRIVER=derelict`


# ACL Drivers

[GoDoc](https://godoc.org/github.com/tg123/docker-wicket/acl)

You can implement your own acl driver and register it with `docker-wicket`. 
For example, adapting to your company's acl system or a MySQL backend.

More drivers, like `ldap`, are on the way. 
PRs are welcomed.

## Built-in Drivers

  * derelict
  
    This driver does nothing but allow any user to access. just for testing purpose.
  

# Index Drivers (v1 only)

## Built-in Drivers

  * mem
  
    store index in memory, would lost after restart. just for testing purpose.
  
  * v1file
  
    Go version of <https://github.com/docker/docker-registry/blob/0.9.1/docker_registry/index.py>.
    store index in json format and is compatible with `docker-registry`'s file storage.
  
