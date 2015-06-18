package htpasswd

import (
	"github.com/docker/docker/pkg/mflag"
	"github.com/tg123/go-htpasswd"

	"gopkg.in/fsnotify.v1"

	"github.com/tg123/docker-wicket/acl"
)

type Driver struct {
	htp *htpasswd.HtpasswdFile
}

func init() {
	d := &Driver{}

	var file string

	mflag.StringVar(&file, []string{"-acl_htpasswd_file"}, "", "File path to htpasswd format file")

	acl.Register("htpasswd", d, func() error {

		htp, err := htpasswd.New(file, htpasswd.DefaultSystems, nil)

		if err != nil {
			return err
		}

		d.htp = htp

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		go func() {
			for {
				event := <-watcher.Events
				if event.Op&fsnotify.Write == fsnotify.Write {
					d.htp.Reload(nil)
				}
			}
		}()

		err = watcher.Add(file)
		if err != nil {
			return err
		}

		return nil
	})
}

func (d *Driver) CanLogin(username acl.Username, password acl.Password) (bool, error) {
	return d.htp.Match(string(username), string(password)), nil
}

// only can access one's own namespace
func (d *Driver) CanAccess(username acl.Username, namespace, repo string, perm acl.Permission) (bool, error) {
	return string(username) == namespace, nil
}
