package util

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

func YamlFileDecode(path string, out interface{}) (err error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, out)
	if err != nil {
		return
	}
	return
}

func YamlFileEncode(path string, in interface{}) (err error) {
	os.Remove(path)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	out, err := yaml.Marshal(in)
	if err != nil {
		return
	}
	_, err = file.Write(out)
	if err != nil {
		return
	}
	return nil
}
