package handler

import (
	"github.com/gocraft/web"

	"github.com/tg123/docker-wicket/acl"
)

type ShareWebContext struct {
}

type RunningContext struct {
	TokenAuth *TokenAuth
	Acl       acl.Driver
}

func Empty(rw web.ResponseWriter, req *web.Request) {
}
