package helper

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/spf13/viper"
)

func TestPhelper(t *testing.T) {
	err := viper.BindEnv("PID")
	if err != nil {
		panic(err)
	}
	pid, err := strconv.Atoi(viper.GetString("PID"))
	if err != nil {
		panic(err)
	}
	p := NewPhelper(uint(pid), PhelperOpts{
		PraseFunc: DefaultProcPraserFunc,
	})
	out, err := p.PraseProc()
	if err != nil {
		panic(err)
	}
	switch t := out.(type) {
	case *PidBindDocker:
		o, err := t.GetDockerUid()

		if err != nil {
			panic(err)
		}
		fmt.Println(o)
	case *PidBindK8sPod:
		o, err := t.GetPodUid()

		if err != nil {
			panic(err)
		}
		fmt.Println(o)

	default:
		fmt.Println(out)
	}

}

func TestCHelper(t *testing.T) {
	err := viper.BindEnv("PID")
	if err != nil {
		panic(err)
	}
	pid, err := strconv.Atoi(viper.GetString("PID"))
	if err != nil {
		panic(err)
	}

}
