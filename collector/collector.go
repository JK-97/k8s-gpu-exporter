package collector

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/spf13/viper"

	"github.com/JK-97/k8s-gpu-exporter/helper"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"tkestack.io/nvml"
)

const (
	namespace = "nvidia_gpu"
)

var labels = []string{"gpu_node", "namepace_name", "gpu_pod_name", "minor_number", "uuid", "name"}

type Collector struct {
	sync.Mutex
	Chelper helper.ContainerHelper

	numGPU                prometheus.Gauge
	fanSpeed              *prometheus.GaugeVec
	powerUsage            *prometheus.GaugeVec
	temperature           *prometheus.GaugeVec
	memoryUtilizationRate *prometheus.GaugeVec
	usedMemory            *prometheus.GaugeVec
	totalMemory           *prometheus.GaugeVec
	freeMemory            *prometheus.GaugeVec
	gpuUtilizationRate    *prometheus.GaugeVec
}

func NewCollector(cHelper helper.ContainerHelper) *Collector {
	return &Collector{
		Chelper: cHelper,
		numGPU: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "num_gpu",
				Help:      "Number of GPU devices",
			},
		),
		fanSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "fan_speed",
				Help:      "Graphics fan speed",
			},
			labels,
		),
		powerUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "power_usage",
				Help:      "Graphics power usage",
			},
			labels,
		),
		temperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperature",
				Help:      "Graphics temperature",
			},
			labels,
		),
		memoryUtilizationRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_utilization_rate",
				Help:      "Graphics memory utilization rate",
			},
			labels,
		),
		usedMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "used_memory",
				Help:      "Graphics used memory ",
			},
			labels,
		),
		totalMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "total_memory",
				Help:      "Graphics used memory ",
			},
			labels,
		),
		freeMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "free_memory",
				Help:      "Graphics used memory ",
			},
			labels,
		),
		gpuUtilizationRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "GPU_utilization_rate",
				Help:      "Graphics utilization rate",
			},
			labels,
		),
	}
}
func failedMsg(msg string, err error) {
	fmt.Printf("%s: %+v\n", msg, err)
}
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	// Only one Collect call in progress at a time.
	c.Lock()
	defer c.Unlock()

	c.fanSpeed.Reset()
	c.powerUsage.Reset()
	c.temperature.Reset()
	c.totalMemory.Reset()
	c.freeMemory.Reset()
	c.usedMemory.Reset()
	c.memoryUtilizationRate.Reset()
	c.gpuUtilizationRate.Reset()

	num, err := nvml.DeviceGetCount()
	if err != nil {
		failedMsg("DeviceGetCount", err)
	} else {
		fmt.Printf("We have %d cards\n", num)
		c.numGPU.Set(float64(num))
		ch <- c.numGPU
	}

	for i := uint(0); i < num; i++ {
		fmt.Printf("GPU-%d ", i)
		dev, err := nvml.DeviceGetHandleByIndex(uint(i))
		if err != nil {
			log.Printf("DeviceHandleByIndex(%d) error: %v", i, err)
			continue
		}

		processes, err := dev.DeviceGetComputeRunningProcesses(32)
		if err != nil {
			log.Printf("DeviceGetComputeRunningProcesses() error: %v", err)
			continue
		} else {
			fmt.Printf("\tDeviceGetComputeRunningProcesses: %d\n", len(processes))
			for _, proc := range processes {
				fmt.Printf("\t\tpid: %d, usedMemory: %d \n", proc.Pid, proc.UsedGPUMemory)
			}
		}

		minorNumber, err := dev.DeviceGetMinorNumber()
		if err != nil {
			log.Printf("MinorNumber() error: %v", err)
			continue
		}
		minor := strconv.Itoa(int(minorNumber))

		uuid, err := dev.DeviceGetUUID()
		if err != nil {
			log.Printf("UUID() error: %v", err)
			continue
		}

		name, err := dev.DeviceGetName()
		if err != nil {
			log.Printf("Name() error: %v", err)
			continue
		}

		// Pod GPU
		podGPUProcessInfos, err := c.Chelper.GetK8sPods(processes)
		if err != nil {
			log.Printf("GetK8sPods() error: %v", err)
		} else {
			for _, podInfo := range podGPUProcessInfos {
				c.usedMemory.WithLabelValues(viper.GetString("NODE_NAME"), podInfo.Pod.Namespace, podInfo.Pod.Name, minor, uuid, name).Set(float64(podInfo.ProcessInfo.UsedGPUMemory))
				fmt.Printf("\t\tnode: %s pod: %s, pid: %d usedMemory: %d \n", viper.GetString("NODE_NAME"), podInfo.Pod.Name, podInfo.ProcessInfo.Pid, podInfo.ProcessInfo.UsedGPUMemory)
			}
		}

		// GPU memory
		freeMemory, usedMemory, totalMemory, err := dev.DeviceGetMemoryInfo()
		if err != nil {
			log.Printf("DeviceGetMemoryInfo() error: %v", err)
		} else {
			c.usedMemory.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(usedMemory))
			c.totalMemory.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(totalMemory))
			c.freeMemory.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(freeMemory))
		}

		// GPU fanspeed
		fanSpeed, err := dev.DeviceGetFanSpeed()
		if err != nil {
			log.Printf("DeviceGetFanSpeed() error: %v", err)
		} else {
			c.fanSpeed.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(fanSpeed))
		}

		// GPU temperature
		temperature, err := dev.DeviceGetTemperature()
		if err != nil {
			log.Printf("DeviceGetTemperature() error: %v", err)
		} else {
			c.temperature.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(temperature))
		}

		// GPU powerUsage
		powerUsage, err := dev.DeviceGetPowerUsage()
		if err != nil {
			log.Printf("DeviceGetPowerUsage() error: %v", err)
		} else {
			c.powerUsage.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(powerUsage))
		}
		// GPU utilization
		utilization, err := dev.DeviceGetUtilizationRates()
		if err != nil {
			log.Printf("DeviceGetUtilizationRates() error: %v", err)
		} else {
			c.gpuUtilizationRate.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(utilization.GPU))
			c.memoryUtilizationRate.WithLabelValues(viper.GetString("NODE_NAME"), "", "", minor, uuid, name).Set(float64(utilization.Memory))
		}

	}

	c.fanSpeed.Collect(ch)
	c.powerUsage.Collect(ch)
	c.temperature.Collect(ch)
	c.totalMemory.Collect(ch)
	c.freeMemory.Collect(ch)
	c.usedMemory.Collect(ch)
	c.memoryUtilizationRate.Collect(ch)
	c.gpuUtilizationRate.Collect(ch)
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.numGPU.Desc()
	c.fanSpeed.Describe(ch)
	c.powerUsage.Describe(ch)
	c.temperature.Describe(ch)
	c.memoryUtilizationRate.Describe(ch)
	c.totalMemory.Describe(ch)
	c.freeMemory.Describe(ch)
	c.usedMemory.Describe(ch)
	c.gpuUtilizationRate.Describe(ch)
}

func K8s() {
	var kubeconfig *string
	kubeconfig = flag.String("kubeconfig", "/home/dev/.kube/config", "absolute path to the kubeconfig file")
	flag.Parse()

	//在 kubeconfig 中使用当前上下文环境，config 获取支持 url 和 path 方式
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 根据指定的 config 创建一个新的 clientset
	clientset, err := kubernetes.NewForConfig(config)
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: "spec.nodeName=dev-ms-7c22",
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the k8s cluster\n", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Printf("pod: %s ,namespace : %s, clusterName %s,id %s \n", pod.Name, pod.Namespace, pod.UID)
		// if pod.Spec.NodeName != "dev-ms-7c22" {
		// 	continue
		// }

		for _, c := range pod.Status.ContainerStatuses {
			fmt.Printf("       %s \n", c.ContainerID)
			fmt.Println(pod.Status.HostIP)
		}
		// for _, c := range pod.Spec.Containers {
		// 	fmt.Printf("       %s \n", c.String())
		// }
	}

}
