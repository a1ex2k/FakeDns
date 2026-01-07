package main

type ApiResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type DomainListResponse struct {
	Domains *[]string `json:"domains"`
}

type AddDomainRequest struct {
	Domains []string `json:"domains"`
}

type RemoveDomainRequest struct {
	Domain string `json:"domain"`
}

type ServiceActionRequest struct {
	Action string `json:"action"`
}
