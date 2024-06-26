package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
)

const (
	DEFAULT_CLIENT_HOST = "localhost"
	DEFAULT_CLIENT_PORT = "8080"
	DEFAULT_SERVER_HOST = "0.0.0.0"
	DEFAULT_SERVER_PORT = "8080"
)

type Config struct {
	Host string
	Port string
}

func DefaultClientConfig() Config {
	return Config{
		Host: DEFAULT_CLIENT_HOST,
		Port: DEFAULT_CLIENT_PORT,
	}
}

func DefaultServerConfig() Config {
	return Config{
		Host: DEFAULT_SERVER_HOST,
		Port: DEFAULT_SERVER_PORT,
	}
}

func WriteConfig(config Config, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	v := reflect.ValueOf(config)
	t := reflect.TypeOf(config)
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := v.Field(i)
		file.WriteString(fmt.Sprintf("%s: %s\n", fieldName, fieldValue))
	}
}

func ReadClientConfig() Config {
	return ReadConfig(DefaultClientConfig())
}

func ReadServerConfig() Config {
	return ReadConfig(DefaultServerConfig())
}

func ReadConfig(defaultConfig Config) Config {
	file, err := os.Open("zerochat.cfg")
	if err != nil {
		log.Println("Creating config file with default values")
		WriteConfig(defaultConfig, "zerochat.cfg")
		return defaultConfig
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			log.Printf("Error: skipping config line %d with wrong format \"%s\"\n", i+1, line)
			continue
		}
		v := reflect.ValueOf(&defaultConfig).Elem()
		search := strings.TrimSpace(parts[0])
		field := v.FieldByName(search)
		if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
			field.SetString(strings.TrimSpace(parts[1]))
		} else {
			log.Printf("Error: skipping config line %d with key not recognized \"%s\"\n", i+1, line)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("err in scanner: %s\n", err)
	}

	log.Printf("Returning config: %#v\n", defaultConfig)
	return defaultConfig
}
