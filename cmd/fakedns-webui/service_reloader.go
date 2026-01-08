package main

import (
	"log"
	"os/exec"
)

type ServiceReloader struct {
	service string
}

func NewServiceReloader(service string) *ServiceReloader {
	return &ServiceReloader{service: service}
}

func (r *ServiceReloader) Stop() error {
	cmd := exec.Command("systemctl", "stop", r.service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("stop output: %s", out)
	}
	return err
}

func (r *ServiceReloader) Reload() error {
	cmd := exec.Command("systemctl", "reload", r.service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("reload output: %s", out)
	}
	return err
}

func (r *ServiceReloader) Restart() error {
	cmd := exec.Command("systemctl", "restart", r.service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("restart output: %s", out)
	}
	return err
}

func (r *ServiceReloader) Start() error {
	cmd := exec.Command("systemctl", "start", r.service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("start output: %s", out)
	}
	return err
}
