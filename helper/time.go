package helper

import (
	. "github.com/go-jet/jet/v2/postgres"
	"time"
)

func CurrentTimeDB() TimestampExpression {
	curTime := Timestamp(time.Now().Year(),
		time.Now().Month(), time.Now().Day(),
		time.Now().Hour(), time.Now().Minute(),
		0)

	return curTime
}
