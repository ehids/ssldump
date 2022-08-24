//go:build !androidgki
// +build !androidgki

/*
Copyright © 2022 CFC4N <cfc4n.cs@gmail.com>

*/
package config

import (
	"os"
	"strings"

	"errors"
)

type PostgresConfig struct {
	eConfig
	PostgresPath string `json:"postgresPath"`
	FuncName     string `json:"funcName"`
}

func NewPostgresConfig() *PostgresConfig {
	config := &PostgresConfig{}
	return config
}

func (this *PostgresConfig) Check() error {

	if this.PostgresPath == "" || len(strings.TrimSpace(this.PostgresPath)) <= 0 {
		return errors.New("Postgres path cant be null.")
	}

	_, e := os.Stat(this.PostgresPath)
	if e != nil {
		return e
	}

	this.FuncName = "exec_simple_query"
	return nil
}
