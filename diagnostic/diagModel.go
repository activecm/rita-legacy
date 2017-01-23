package diagnostic

import "time"

type diagModel struct {
	ErrorMessage string      `json:"errorMessage" bson:"errorMessage"`
	Time         time.Time   `json:"time" bson:"time"`
	FailedLog    interface{} `json:"failedLog" bson:"failedLog"`
}
