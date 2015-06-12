package acl

import (
	"fmt"
)

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
