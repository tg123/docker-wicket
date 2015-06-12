package mem

import (
	"fmt"

	"github.com/robfig/go-cache"
	"github.com/tg123/docker-wicket/index"
)

// only for dev purpose

type Driver struct {
}

var mem = cache.New(0, 0) // just magic because this driver is only for dev

func init() {

	d := &Driver{}

	index.Register("mem", d, func() error { return nil })
}

func key(namespace, repo string) string {
	return fmt.Sprintf("%v/%v", namespace, repo)
}

func (d *Driver) GetIndexImages(namespace, repo string) ([]index.Image, error) {

	m, ok := mem.Get(key(namespace, repo))

	if !ok {
		m = make([]index.Image, 0)
	}

	return m.([]index.Image), nil
}

func (d *Driver) UpdateIndexImages(namespace, repo string, images []index.Image) error {
	mem.Set(key(namespace, repo), images, -1)

	return nil
}

func (d *Driver) CreateRepo(namespace, repo string) error {
	return nil
}

func (d *Driver) DeleteRepo(namespace, repo string) error {
	return nil
}
