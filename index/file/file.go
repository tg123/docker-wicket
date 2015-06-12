package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/docker/pkg/mflag"
	"github.com/tg123/docker-wicket/index"
)

type Driver struct {
	Path string
}

func init() {

	d := &Driver{}

	mflag.StringVar(&d.Path, []string{"-v1_index_file_path"}, "", "Path to v1 repo")

	index.Register("v1file", d, func() error {
		if d.Path == "" {
			return fmt.Errorf("path to v1 repo not set")
		}

		return nil
	})
}

func (d *Driver) repoPath(namespace, repo string) string {
	return fmt.Sprintf("%v/repositories/%v/%v", d.Path, namespace, repo)
}

func (d *Driver) indexFile(namespace, repo string) string {
	return fmt.Sprintf("%v/_index_images", d.repoPath(namespace, repo))
}

func (d *Driver) GetIndexImages(namespace, repo string) ([]index.Image, error) {

	// repositories/library/test/_index_images

	m := make([]index.Image, 0)

	f := d.indexFile(namespace, repo)

	if _, err := os.Stat(f); os.IsNotExist(err) {
		return m, nil
	}

	b, err := ioutil.ReadFile(f)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &m)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (d *Driver) UpdateIndexImages(namespace, repo string, images []index.Image) error {
	b, err := json.Marshal(images)

	if err != nil {
		return err
	}

	p := d.repoPath(namespace, repo)

	err = os.MkdirAll(p, 0755)

	if err != nil {
		return err
	}

	f := d.indexFile(namespace, repo)

	return ioutil.WriteFile(f, b, 0644)
}

func (d *Driver) CreateRepo(namespace, repo string) error {
	return nil
}

func (d *Driver) DeleteRepo(namespace, repo string) error {
	return nil
}
