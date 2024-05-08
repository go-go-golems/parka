package utils

import "fmt"

// NoPageFoundError is returned by Render if no template was found, in which case
// the Render is skipped and moves on to the next middleware.
type NoPageFoundError struct {
	Page string
}

func (e *NoPageFoundError) Error() string {
	return fmt.Sprintf("no page found for %s", e.Page)
}
