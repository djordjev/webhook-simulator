package mapping

import (
	"encoding/json"
	"fmt"
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"io/fs"
	"log"
	"path/filepath"
	"sync"
)

const Root = "."

type mapping struct {
	config     config.Config
	fileSystem fs.FS
	mappings   []Flow
	lock       sync.Mutex
}

func (m *mapping) Refresh() (err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	result := make(chan *Flow)
	counter := 0
	m.mappings = make([]Flow, 0)

	err = fs.WalkDir(m.fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if path == Root {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".whs" {
			return nil
		}

		counter += 1
		go m.readMapping(path, result)

		return nil
	})

	for i := 0; i < counter; i++ {
		flow := <-result
		if flow != nil {
			m.mappings = append(m.mappings, *flow)
		}
	}

	return
}

func (m *mapping) readMapping(path string, result chan<- *Flow) {
	log.Println(fmt.Sprintf("reading file %s", path))

	var flow *Flow
	defer func() {
		result <- flow
	}()

	data, err := fs.ReadFile(m.fileSystem, path)
	if err != nil {
		log.Println(fmt.Sprintf("unable to read content of file %s", path))
		return
	}

	err = json.Unmarshal(data, &flow)
	if err != nil {
		log.Println(fmt.Sprintf("unable to parse content of file %s", path))
		return
	}
}

func (m *mapping) GetMappings() []Flow {
	return m.mappings
}

func NewMapping(config config.Config, fs fs.FS) Mapper {
	return &mapping{config: config, fileSystem: fs}
}
