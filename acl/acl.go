package acl

type Username string
type Password string

type Permission int

const (
	READ Permission = iota
	WRITE
	DELETE
)

type Driver interface {
	CanLogin(username Username, password Password) (bool, error)

	CanAccess(username Username, namespace, repo string, perm Permission) (bool, error)
}
