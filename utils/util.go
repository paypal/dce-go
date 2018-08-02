package utils

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
)

type ConditionFunc func() (string, error)

func PluginPanicHandler(condition ConditionFunc) (res string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recover : %v \n", r)
			err = errors.New(fmt.Sprintf("Recover : %v \n", r))
		}
	}()

	if res, err = condition(); err != nil {
		log.Errorf("Error executing plugins: %v \n", err)
		return res, err
	}
	return res, err
}


func GetDceLogFileDescriptor (filename string) (*os.File, error) {

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return file, err
}