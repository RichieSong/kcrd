package kcrd

import (
	"fmt"
	informer "go/src/kcrd/pkg/client/informers/externalversions/stable/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"time"
)

type Controller struct {
	informer informer.CronTabInformer
	queue    workqueue.RateLimitingInterface
}

func NewController(informer informer.CronTabInformer) *Controller {
	return &Controller{
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "crontab-controller"),
		informer: informer,
	}
}
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown()
	klog.Info("Starting Pod controller")

	if !cache.WaitForCacheSync(stopCh, c.informer.Informer().HasSynced) {
		return fmt.Errorf("timeout waiting for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	<-stopCh
	klog.Info("Stopping crontab controller")
	return nil

}

func (c *Controller) runWorker() {
	//for
}

// 实现业务逻辑

func (c *Controller)processNextItem()bool  {
	key,quit:= c.queue.Get()
	if quit{
		return false
	}
	defer c.queue.Done(key)
	//err:=c.syncTo
	return true
}