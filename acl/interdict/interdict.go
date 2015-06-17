package interdict

import (
	"github.com/tg123/docker-wicket/acl"
)

type Driver struct {
}

func init() {
	d := &Driver{}

	acl.Register("interdict", d, func() error { return nil })
}

func (d *Driver) CanLogin(username acl.Username, password acl.Password) (bool, error) {
	return false, nil
}

func (d *Driver) CanAccess(username acl.Username, namespace, repo string, perm acl.Permission) (bool, error) {
	return false, nil
}
