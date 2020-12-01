package kcrd

import (
	"flag"
	"go/src/kcrd/pkg/client/clientset/versioned"
	"go/src/kcrd/pkg/client/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sync"
)
var config *rest.Config
var (
	clsMap  = make(map[string]*kubernetes.Clientset)
	clsLock = &sync.Mutex{}
)
func CLS(env string) *kubernetes.Clientset {
	clsLock.Lock()
	defer clsLock.Unlock()
	return clsMap[env]
}

// buildConfigFromFlagContext 根据context和配置文件创建config
func buildConfigFromFlagContext(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}



func InitK8s() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// use the current context in kubeconfig
	var err error
	// config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clsLock.Lock()
	defer clsLock.Unlock()
	for _, env := range []string{"PRE","DEV"} {
		config, err = buildConfigFromFlagContext(env, *kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		clsMap[env] = clientset
	}
}

func setupSignalHander() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c:= make(chan os.Signal,2)
	signal.Notify(c,[]os.Signal{os.Interrupt,syscall.SIGTERM}...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1)
	}()
	return
}
func main() {
	flag.Parse()
	//_ := CLS("PRE")
	InitK8s()
	klog.Info(config)
	// 实例化crontab
	crontabClientSet,err:=versioned.NewForConfig(config)
	if err!=nil{
		klog.Fatalf( "Error init kubernetes contab client: %s",err.Error())
	}
	stopCh :=setupSignalHander()
	// 实例化crontab former工厂类
	sharedInformerFactory:=externalversions.NewSharedInformerFactory(crontabClientSet,time.Minute)
	// 启动informer 监听list
	go sharedInformerFactory.Start(stopCh)

	// 实例化crontab控制器

	controller:= NewController(sharedInformerFactory.Stable().V1beta1().CronTabs())
	// 启动控制器循环
	if err:= controller.Run(1,stopCh);err!=nil{
		klog.Fatalf("%s",err.Error)
	}
}

