package updating

import (
	"github.com/djordjev/webhook-simulator/internal/packages/config"
	"github.com/djordjev/webhook-simulator/internal/packages/mapping"
	"github.com/fsnotify/fsnotify"
	"log"
	"strings"
)

type Updater interface {
	Listen()
}

type FSNotifyUpdater struct {
	mapper mapping.Mapper
	config config.Config
}

func (f FSNotifyUpdater) Listen() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil || watcher == nil {
		log.Println("unable to listen directory")
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				{
					if !ok {
						return
					}

					isMappingFile := strings.HasSuffix(event.Name, ".whs")

					isWrite := event.Has(fsnotify.Write)
					isCreate := event.Has(fsnotify.Create)
					isRename := event.Has(fsnotify.Rename)
					isDelete := event.Has(fsnotify.Remove)

					log.Println("isWrite", isWrite, "isCreate", isCreate, "isRename", isRename, "isDelete", isDelete)

					listeningType := isWrite || isCreate || isRename || isDelete

					if isMappingFile && listeningType {
						e := f.mapper.Refresh()
						if e != nil {
							log.Println("failed to refresh due to mapping folder change")
						}
					}

				}

			case e, ok := <-watcher.Errors:
				{
					if !ok {
						return
					}

					log.Println("error: ", e)
				}
			}
		}
	}()

	err = watcher.Add(f.config.Mapping)

}

func NewUpdater(mapper mapping.Mapper, cfg config.Config) Updater {
	return FSNotifyUpdater{
		mapper: mapper,
		config: cfg,
	}
}
