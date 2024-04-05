package repository

import "fmt"

type PublishUpdateError struct{}

func (e PublishUpdateError) Error() string {
	return fmt.Sprintf("Update publishing has failed")
}
