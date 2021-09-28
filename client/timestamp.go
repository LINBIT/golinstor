package client

import (
	"encoding/json"
	"strconv"
	"time"
)

type TimeStampMs struct {
	time.Time
}

func (t *TimeStampMs) UnmarshalJSON(s []byte) (err error) {
	r := string(s)
	q, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	t.Time = time.Unix(q/1000, (q%1000)*1_000_000)
	return nil
}

func (t TimeStampMs) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Unix() * 1000)
}
