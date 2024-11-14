package webpush

import (
	"fmt"
	"strconv"
	"time"
)

type StringerValidator interface {
	String() string
	Validate() error
}

type Epoch struct {
	time.Time
}

func (e Epoch) MarshalJSON() ([]byte, error) {
	ts := e.Unix()

	return []byte(fmt.Sprintf("%d", ts)), nil
}

func (e *Epoch) UnmarshalJSON(val []byte) (err error) {
	var epoch int64
	stringified := string(val)

	if epoch, err = strconv.ParseInt(stringified, 10, 64); err != nil {
		return
	}

	(*e).Time = time.Unix(epoch, 0)

	return
}
