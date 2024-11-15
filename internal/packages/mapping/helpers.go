package mapping

import "strings"

var supportedExtensions = [...]string{".whs", ".json"}

func HasMappingFileExtension(name string) bool {
	for i := 0; i < len(supportedExtensions); i++ {
		if strings.HasSuffix(name, supportedExtensions[i]) {
			return true
		}
	}
	return false
}
