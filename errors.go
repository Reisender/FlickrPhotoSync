package photosync

import (
	"fmt"
)

// API Error type
type Error struct {
	response string
}
func (e Error) Error() string {
	return fmt.Sprintf("API fail: %s", e.response)
}

