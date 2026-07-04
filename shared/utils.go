package shared

import "log"

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s %s\n", err, msg)
	}
}
