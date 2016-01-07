package util

import (
	"github.com/Sirupsen/logrus"
	"os"
	"strings"
)

func NewLogger(name string) (logger *logrus.Logger) {
	logger = logrus.New()
	logOutput, err := os.OpenFile(GetBasePath()+"/log/"+name+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	logger.Out = logOutput
	logger.Formatter = new(logrus.JSONFormatter)
	return logger
}

func GetBasePath() string {
	// @todo small hack
	dir, _ := os.Getwd()
	split := strings.Split(dir, "gotrade")
	return split[0] + "/gotrade"
}
