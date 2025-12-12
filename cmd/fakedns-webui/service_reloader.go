package main

import (
	"log"
	"os/exec"
)

// ServiceReloader отвечает ТОЛЬКО за reload systemd-сервиса
type ServiceReloader struct {
	service string
}

func NewServiceReloader(service string) *ServiceReloader {
	return &ServiceReloader{service: service}
}

func (r *ServiceReloader) Reload() error {
	cmd := exec.Command("systemctl", "reload", r.service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("reload output: %s", out)
	}
	return err
}
