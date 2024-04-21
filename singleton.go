package sabot

import "github.com/pkg/errors"

var singleton *Sabot

func (sabot *Sabot) Singleton() {
	singleton = sabot
}

func Singleton() (sabot *Sabot, err error) {

	sabot = singleton

	if singleton == nil {
		err = errors.Errorf("singleton logger is undefined")
	}

	return
}
