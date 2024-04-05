package repository

import "fmt"

type Keys struct{}

var PubSubChannel = "channel"

func (k Keys) GetNotifyChannel() string {
	return fmt.Sprintf("%s:%s:", NotifyPrefix, PubSubChannel)
}
