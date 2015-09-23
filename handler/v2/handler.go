package v2

// https://github.com/cesanta/docker_auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gocraft/web"

	"github.com/tg123/docker-wicket/handler"

	"github.com/tg123/docker-wicket/acl"
)

type RunningContext struct {
	handler.RunningContext
}

type context struct {
	*handler.ShareWebContext

	namespace string
	repo      string

	permsWant []string

	authReq handler.AuthRequest
}

var runningContext *RunningContext

// TODO check docker sourse code
var accessMap = map[string]acl.Permission{
	"pull": acl.READ,
	"push": acl.WRITE,
}

func (c *context) commonHeader(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	rw.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	next(rw, req)
}

// auth_server.server.ParseRequest

func (c *context) parseRequest(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {

	// GET /v2/token/?service=registry.docker.com&scope=repository:samalba/my-app:push&account=jlhawn HTTP/1.1
	c.authReq.Account = req.FormValue("account")

	c.authReq.Service = req.FormValue("service")

	scope := req.FormValue("scope")

	if scope != "" {
		parts := strings.Split(scope, ":")
		if len(parts) != 3 {
			http.Error(rw, fmt.Sprintf("invalid scope: %q", scope), http.StatusBadRequest)
			return
		}

		c.authReq.Type = parts[0]
		c.authReq.Name = parts[1]

		if strings.Contains(parts[1], "/") {
			nr := strings.SplitN(parts[1], "/", 2)

			c.namespace = nr[0]
			c.repo = nr[1]
		} else {
			c.namespace = "library"
			c.repo = parts[1]
		}

		c.permsWant = strings.Split(parts[2], ",")
	}

	next(rw, req)
}

func (c *context) authAccess(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	username, password, ok := req.BasicAuth()

	if c.authReq.Account != "" && c.authReq.Account != username {
		http.Error(rw, "account is not same as login user", http.StatusForbidden)
		return
	}

	var _username acl.Username

	if ok {
		_username = acl.Username(username)
	} else {
		_username = acl.Anonymous
	}

	ok, err := runningContext.Acl.CanLogin(_username, acl.Password(password))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if !ok {

		if _username == acl.Anonymous {
			http.Error(rw, "", http.StatusUnauthorized)
		} else {
			http.Error(rw, "", http.StatusForbidden)
		}

		return
	}

	// check actions
	for _, v := range c.permsWant {

		p := accessMap[v]

		ok, err := runningContext.Acl.CanAccess(_username, c.namespace, c.repo, p)

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		if ok {
			c.authReq.Actions = append(c.authReq.Actions, v)
		}
	}

	sort.Strings(c.authReq.Actions)

	next(rw, req)
}

func (c *context) writeToken(rw web.ResponseWriter, req *web.Request) {

	token, err := runningContext.TokenAuth.CreateToken(&c.authReq)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")

	result, err := json.Marshal(&map[string]string{"token": token})

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Write(result)
}

func InstallHandler(rootRouter *web.Router, rc *RunningContext) {

	runningContext = rc

	c := context{}

	v2 := rootRouter.Subrouter(c, "/v2").
		Middleware((*context).commonHeader)

	v2.Subrouter(c, "/token").
		Middleware((*context).parseRequest).
		Middleware((*context).authAccess).
		Get("/", (*context).writeToken)
}
