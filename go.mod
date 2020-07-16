module github.com/JK-97/k8s-gpu-exporter

go 1.14

require (
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/viper v1.7.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	sigs.k8s.io/yaml v1.2.0 // indirect
	tkestack.io/nvml v0.0.0-00010101000000-000000000000
)

replace tkestack.io/nvml => github.com/tkestack/go-nvml v0.0.0-20191217064248-7363e630a33e
