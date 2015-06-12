package index

import (
	"fmt"
)

type Image struct {
	Id       string `json:"id"`
	Checksum string `json:"checksum,omitempty"`
}

type Driver interface {
	GetIndexImages(namespace, repo string) ([]Image, error)

	UpdateIndexImages(namespace, repo string, images []Image) error

	CreateRepo(namespace, repo string) error

	DeleteRepo(namespace, repo string) error
}

type managedDriver struct {
	driver Driver
	check  func() error
}

var drivers = make(map[string]managedDriver)

func Load(name string) (Driver, error) {

	d, ok := drivers[name]

	if !ok {
		return nil, fmt.Errorf("Driver not found")
	}

	if err := d.check(); err != nil {
		return nil, err
	}

	return d.driver, nil
}

func Register(name string, driver Driver, check func() error) {
	drivers[name] = managedDriver{driver, check}
}
