package v1

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gocraft/web"

	"github.com/tg123/docker-wicket/handler"

	"github.com/tg123/docker-wicket/acl"
	"github.com/tg123/docker-wicket/index"
)

type RunningContext struct {
	handler.RunningContext

	Endpoints string
	Index     index.Driver
}

type permission struct {
	acl.Permission

	name string
}

var runningContext *RunningContext

var accessMap = map[string]permission{
	"GET":    permission{acl.READ, "read"},
	"PUT":    permission{acl.WRITE, "write"},
	"DELETE": permission{acl.DELETE, "delete"},
}

type context struct {
	*handler.ShareWebContext

	namespace string
	repo      string
}

func (c *context) checkSignature(namespace, repo, signature, access string) bool {

	return runningContext.TokenAuth.Verify(signature, func(resourceActions handler.ResourceActions) error {

		for _, r := range resourceActions {

			if r.Name == fmt.Sprintf("%v/%v", namespace, repo) {
				for _, a := range r.Actions {
					if a == access {
						return nil
					}
				}
			}
		}

		return fmt.Errorf("No match access")

	}) == nil
}

func (c *context) commonHeader(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	rw.Header().Set("X-Docker-Registry-Version", "0.9.1")
	next(rw, req)
}

// ping

func (c *context) ping(rw web.ResponseWriter, req *web.Request) {
	rw.Header().Set("X-Docker-Registry-Standalone", "false")
	http.Error(rw, "true", http.StatusOK)
}

//

func (c *context) authAccess(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	username, password, ok := req.BasicAuth()

	if ok {

		a, ok := accessMap[req.Method]

		if !ok {
			http.Error(rw, "", http.StatusMethodNotAllowed)
			return
		}

		ok, err := runningContext.Acl.CanLogin(acl.Username(username), acl.Password(password))

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		if !ok {
			http.Error(rw, "", http.StatusForbidden)
			return
		}

		ok, err = runningContext.Acl.CanAccess(acl.Username(username), c.namespace, c.repo, a.Permission)

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		if !ok {
			http.Error(rw, "", http.StatusForbidden)
			return
		}

		next(rw, req)
		return
	}

	// Authorization: Token signature=123,repository="library/test",access=write
	a, ok := req.Header["Authorization"]

	if ok {
		m := make(map[string]string)

		s := strings.TrimLeft(a[0], "Token ")

		for _, p := range strings.Split(s, ",") {

			kv := strings.Split(p, "=")

			k := kv[0]
			v := kv[1]

			m[k] = v
		}

		nr := strings.SplitN(strings.Trim(m["repository"], `"`), "/", 2)

		m["namespace"] = nr[0]
		m["repo"] = nr[1]

		if c.checkSignature(nr[0], nr[1], m["signature"], m["access"]) {
			next(rw, req)
			return
		}

	}

	http.Error(rw, "", http.StatusUnauthorized)
}

func (c *context) readNamespace(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	ns, ok := req.PathParams["namespace"]

	if !ok {
		ns = "library"
	}

	c.namespace = ns

	c.repo = req.PathParams["repo"]

	next(rw, req)
}

func (c *context) generateToken(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {

	if req.Header.Get("X-Docker-Token") == "true" {

		a, ok := accessMap[req.Method]

		if !ok {
			http.Error(rw, "", http.StatusMethodNotAllowed)
			return
		}

		sig, err := runningContext.TokenAuth.CreateToken(&handler.AuthRequest{
			Name:    fmt.Sprintf("%v/%v", c.namespace, c.repo),
			Actions: []string{a.name},
			Service: runningContext.TokenAuth.Service,
		})

		if !ok {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		t := fmt.Sprintf(`signature=%v,repository="%v/%v",access=%v`, sig, c.namespace, c.repo, a.name)

		rw.Header().Set("X-Docker-Endpoints", runningContext.Endpoints)
		rw.Header().Set("WWW-Authenticate", "Token "+t)
		rw.Header().Set("X-Docker-Token", t)
	}

	next(rw, req)
}

func (c *context) sayAccess(rw web.ResponseWriter, req *web.Request) {
	rw.Write([]byte(`{"access": true}`))
}

func (c *context) getImages(rw web.ResponseWriter, req *web.Request) {

	rw.Header().Set("Content-Type", "application/json")

	m, err := runningContext.Index.GetIndexImages(c.namespace, c.repo)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(m)

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	rw.Write(b)
}

func (c *context) updateImageIndex(req *web.Request) error {
	images, err := runningContext.Index.GetIndexImages(c.namespace, c.repo)
	if err != nil {
		return err
	}

	newImages := make([]index.Image, 0)

	b, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &newImages)

	if err != nil {
		return err
	}

	images = append(images, newImages...)

	m := make(map[string]index.Image)

	for _, i := range images {

		if _, ok := m[i.Id]; !ok {
			m[i.Id] = i
		} else if m[i.Id].Checksum == "" {
			m[i.Id] = i
		}
	}

	images = make([]index.Image, len(m))

	i := 0
	for _, v := range m {
		images[i] = v
		i++
	}

	return runningContext.Index.UpdateIndexImages(c.namespace, c.repo, images)
}

func (c *context) createImages(rw web.ResponseWriter, req *web.Request) {

	if err := c.updateImageIndex(req); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	http.Error(rw, "", http.StatusNoContent)
}

func (c *context) createRepo(rw web.ResponseWriter, req *web.Request) {

	if err := c.updateImageIndex(req); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	http.Error(rw, "", http.StatusOK)
}

func (c *context) deleteRepo(rw web.ResponseWriter, req *web.Request) {

	if err := runningContext.Index.DeleteRepo(c.namespace, c.repo); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	http.Error(rw, "", http.StatusNoContent)
}

func InstallHandler(rootRouter *web.Router, rc *RunningContext) {

	runningContext = rc

	c := context{}

	v1 := rootRouter.Subrouter(c, "/v1").
		Middleware((*context).commonHeader).
		Get("/", (*context).ping).
		Get("/_ping", (*context).ping)

	v1.Subrouter(c, "/users").
		Middleware((*context).authAccess).
		Get("/", handler.Empty).
		Post("/", handler.Empty).
		Put("/", handler.Empty)

	v1.Subrouter(c, "/repositories").
		Middleware((*context).readNamespace).
		Middleware((*context).authAccess).
		Middleware((*context).generateToken).
		// auth
		Put("/:repo/auth", handler.Empty).
		Put("/:namespace/:repo/auth", handler.Empty).
		// images
		Get("/:repo/images", (*context).getImages).
		Get("/:namespace/:repo/images", (*context).getImages).
		Put("/:repo/images", (*context).createImages).
		Put("/:namespace/:repo/images", (*context).createImages).
		// layer
		Get("/:repo/layer/:image/access", (*context).sayAccess).
		Get("/:namespace/:repo/layer/:image/access", (*context).sayAccess).
		// repo
		Put("/:repo", (*context).createRepo).
		Put("/:namespace/:repo/", (*context).createRepo).
		Delete("/:repo/", (*context).deleteRepo).
		Delete("/:namespace/:repo", (*context).deleteRepo)
}
