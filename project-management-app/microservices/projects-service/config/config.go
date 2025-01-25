package config

import "os"

type Config struct {
	Address                 string
	JaegerAddress           string
	UsersServiceAddress string
	TasksServiceAddress string
}

func GetConfig() Config {
	return Config{
		Address:                 os.Getenv("ORDERING_SERVICE_ADDRESS"),
		JaegerAddress:           os.Getenv("JAEGER_ADDRESS"),
		UsersServiceAddress: os.Getenv("USERS_SERVICE_ADDRESS"),
		TasksServiceAddress: os.Getenv("TASKS_SERVICE_ADDRESS"),

	}
}
