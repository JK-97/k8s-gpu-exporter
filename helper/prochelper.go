package helper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"tkestack.io/nvml"
)

// ubuntu 20.04
// k8s proc /kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-poddc380811_8168_4a9e_9b4a_56ded1b6fe9c.slice/docker-fa7f0ec4fbcbb70c69565a5b2d269926e04ec5601dff59235199d6e60198e95c.scope
// docker proc /system.slice/docker-293f758723d0652a0b9fa869106e600079e56dae362ca84de12942d38524a272.scope
// native proc /
// centos
///kubepods/besteffort/pod762f2fe7-8a9d-450a-81e7-88094fc057d0/557a3158f637c3021ecba8450ee945570bdb66876922afafa5dd6e0b35131f71
// k8s proc  /kubepods/pod8a0412cb-ae87-4bd5-b49d-690fd86f942e/6497d9f440dad7c3d432100ab3c5e895831549467d81eeef176bdec12da43fd9
// docker proc /docker/04807c588b00fbf639c1e4896eab5a076771b6905b60f9ccc56bf6984ba9b71a
// native proc /

type PidBindDocker struct {
	dockerUid string
}

func (o *PidBindDocker) GetDockerUid() (string, error) {
	return o.dockerUid, nil
}

func (o *PidBindDocker) GetPodUid() (string, error) {
	return "", errors.New("pid in native docker ,have not k8s uid")
}
func (o *PidBindDocker) SetDockerUid(uid string) error {
	o.dockerUid = uid
	return nil
}

func (o *PidBindDocker) SetPodUid(uid string) error {
	return errors.New("pid in native docker ,have not k8s uid")
}

// check interface impelement
var _ PidPraseOut = new(PidBindDocker)

type PidBindK8sPod struct {
	dockerUid string
	podUid    string
}

func (o *PidBindK8sPod) GetPodUid() (string, error) {
	return o.podUid, nil
}

func (o *PidBindK8sPod) GetDockerUid() (string, error) {
	return o.dockerUid, nil
}

func (o *PidBindK8sPod) SetPodUid(uid string) error {
	o.podUid = uid
	return nil
}

func (o *PidBindK8sPod) SetDockerUid(uid string) error {
	o.dockerUid = uid
	return nil
}

// check interface impelement
var _ PidPraseOut = new(PidBindK8sPod)

type Phelper struct {
	ProcHelper
	pid    uint
	praser ProcPraseFunc
}

type PhelperOpts struct {
	PraseFunc ProcPraseFunc
}

func NewPhelper(pid uint, opt PhelperOpts) *Phelper {
	return &Phelper{
		pid:    pid,
		praser: opt.PraseFunc,
	}
}

func DefaultProcPraserFunc(procInfo string) (PidPraseOut, error) {
	if procInfo == "/" {
		return nil, errors.New("cat not prase native pid")
	}

	if strings.Contains(string(procInfo), "docker") && !strings.HasPrefix(procInfo, "/kubepods") {
		out := new(PidBindDocker)
		reDocker, err := regexp.Compile("docker-(.*).scope")
		if err != nil {
			return nil, err
		}
		reDockerOut := reDocker.FindStringSubmatch(procInfo)
		if len(reDockerOut) != 2 {
			return nil, fmt.Errorf("Prase docker  error: %s ,but out %v", procInfo, reDockerOut)
		}

		err = out.SetDockerUid(reDockerOut[1])
		if err != nil {
			return nil, err
		}
		return out, nil

	}
	if strings.HasPrefix(procInfo, "/kubepods") {
		out := new(PidBindK8sPod)

		reDocker, err := regexp.Compile("docker-(.*).scope")
		if err != nil {
			return nil, err
		}
		reDockerOut := reDocker.FindStringSubmatch(procInfo)
		if len(reDockerOut) != 2 {
			return nil, fmt.Errorf("Prase docker  error: %s ,but out %v", procInfo, reDockerOut)
		}
		err = out.SetDockerUid(reDockerOut[1])
		if err != nil {
			return nil, err
		}

		reK8s, err := regexp.Compile(".*pod(.*).slice/docker")
		if err != nil {
			return nil, err
		}
		reK8sOut := reK8s.FindStringSubmatch(procInfo)
		if len(reDockerOut) != 2 {
			return nil, fmt.Errorf("Prase k8s  error: %s ,but out %v", procInfo, reK8sOut)
		}
		replaceOut := strings.ReplaceAll(reK8sOut[1], "_", "-")
		err = out.SetPodUid(replaceOut)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	return nil, errors.New("Prase error : Unknow Proc Type")
}

func (p *Phelper) PraseProc() (PidPraseOut, error) {
	if p.praser == nil {
		p.praser = DefaultProcPraserFunc
		fmt.Println("Not set PraseFunc in ops ,use the DefaultProcPraserFunc")
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/cpuset", p.pid))
	if err != nil {
		return nil, err
	}
	procInfo := strings.TrimSpace(string(data))

	return p.praser.Prase(procInfo)
}

// check interface impelement
var _ ProcHelper = new(Phelper)

type CHelper struct {
	ContainerHelper
	KClient  *kubernetes.Clientset
	PraseFuc ProcPraseFunc
}
type CHelperOps struct {
	KClient  *kubernetes.Clientset
	PraseFuc ProcPraseFunc
}

func NewCHepler(ops *CHelperOps) *CHelper {
	return &CHelper{
		KClient:  ops.KClient,
		PraseFuc: ops.PraseFuc,
	}
}

func (c *CHelper) GetPodMap() (map[string]v1.Pod, error) {
	pods, err := c.KClient.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", viper.GetString("NODE_NAME")),
	})
	if err != nil {
		return nil, err
	}

	podMap := make(map[string]v1.Pod)
	for _, pod := range pods.Items {
		podMap[string(pod.UID)] = pod
	}
	return podMap, nil
}

func (c *CHelper) GetK8sPods(processesInfo []*nvml.ProcessInfo) ([]*PodGPUInfo, error) {
	podMap, err := c.GetPodMap()
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	res := make([]*PodGPUInfo, 0)
	for _, pi := range processesInfo {
		ph := NewPhelper(pi.Pid, PhelperOpts{PraseFunc: c.PraseFuc})
		out, err := ph.PraseProc()
		if err != nil {
			return nil, err
		}
		switch t := out.(type) {
		case *PidBindK8sPod:
			podUid, err := t.GetPodUid()
			if err != nil {
				return nil, err
			}
			if pod, ok := podMap[podUid]; ok {
				res = append(res, &PodGPUInfo{
					ProcessInfo: pi,
					Pod:         pod,
				})
			}
			return res, nil
		default:
			continue
		}
	}

	return nil, nil
}

func (c *CHelper) GetContainers(processInfo []*nvml.ProcessInfo) ([]*types.Container, error) {
	return nil, nil
}

// check interface impelement
var _ ContainerHelper = new(CHelper)
