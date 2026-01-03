package assert

import "log"

func Success[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}
