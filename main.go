package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JK-97/k8s-gpu-exporter/helper"

	"github.com/JK-97/k8s-gpu-exporter/collector"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"tkestack.io/nvml"
)

var (
	nvidiaDockerNvmlLink = "/usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1"
	nvmlLibLink          = "/usr/lib/x86_64-linux-gnu/libnvidia-ml.so"
)

var (
	addr       = flag.String("address", ":9445", "Address to listen on for web interface and telemetry.")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file, default get config from pod binding ServiceAccount.")
)

func init() {
	autoLoadNvml()
	err := viper.BindEnv("NODE_NAME")
	if err != nil {
		panic(err)
	}
	ver, err := nvml.SystemGetDriverVersion()
	if err != nil {
		panic(fmt.Errorf("SystemGetDriverVersion %s", err))
	} else {
		fmt.Printf("SystemGetDriverVersion: %s\n", ver)
	}
}

func main() {
	defer nvml.Shutdown()
	flag.Parse()
	if !flag.Parsed() {
		panic(errors.New("Has not prase command line"))
	}
	config, err := GetK8sConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	cHelper := helper.NewCHepler(&helper.CHelperOps{
		KClient:   clientset,
		PraseFunc: helper.DefaultProcPraserFunc,
	})

	prometheus.MustRegister(collector.NewCollector(cHelper))

	fmt.Println("Handle URL path: /metrics")
	fmt.Printf("Listen on %v\n", *addr)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatalf("ListenAndServe error: %v", http.ListenAndServe(*addr, nil))
}

func GetK8sConfig() (*rest.Config, error) {
	if *kubeconfig == "" {
		fmt.Println("Not specify a config ,use default config from k8s env")
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

func autoLoadNvml() {
	_, err := os.Stat(nvidiaDockerNvmlLink)
	if err != nil {
		panic(err)
	} else {
		dst, err := os.Readlink(nvidiaDockerNvmlLink)
		if err != nil {
			panic(err)
		}
		_, err = os.Stat(nvmlLibLink)
		if err == nil {
			os.Remove(nvmlLibLink)
		}
		err = os.Symlink(dst, nvmlLibLink)
		if err != nil {
			panic(err)
		}
	}
	if err = nvml.Init(); err != nil {
		panic(err)
	}

}
