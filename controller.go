package main

import (
	"fmt"
	"go/src/kcrd/pkg/apis/stable/v1beta1"
	informer "go/src/kcrd/pkg/client/informers/externalversions/stable/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	c := &Controller{
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "crontab-controller"),
		informer: informer,
	}
	klog.Infof("Setting up crontab controller")
	// 注册监听事件
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})
	return c
}
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown()
	klog.Info("Starting Pod controller")

	if !cache.WaitForCacheSync(stopCh, c.informer.Informer().HasSynced) {
		return fmt.Errorf("timeout waiting for caches to sync")
	}

	klog.Info("cache sync success")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	<-stopCh
	klog.Info("Stopping crontab controller")
	return nil

}

func (c *Controller) runWorker() {
}

// 实现业务逻辑
func (c *Controller) processNextItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	// 告诉队列 我们已经完成处理的此key
	// 这将为其他worker解锁该key
	// 这将确保安全的并行处理，因为永远不会并行处理具有相同key的两个pod
	defer c.queue.Done(obj)
	// 调用业务逻辑处理方法
	err := func(obj interface{}) error {
		// 告诉队列 我们已经完成处理的此key
		// 这将为其他worker解锁该key
		// 这将确保安全的并行处理，因为永远不会并行处理具有相同key的两个pod
		defer c.queue.Done(obj)
		var ok bool
		var key string
		if key, ok = obj.(string); !ok {
			c.queue.Forget(obj)
			return fmt.Errorf("expected string in workqueueu but get %v", obj)
		}
		// 业务逻辑处理
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("sync error: %v", err)
		}
		c.queue.Forget(obj)
		klog.Info("successfully syncd %s", key)
		return nil
	}(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	return true
}

// key - > crontab - > indexer
func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// 获取crontab 实际上是从indexer获取的
	crontab, err := c.informer.Lister().CronTabs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// 对于的crontab对象已经被删除了
			klog.Warningf("Crontab deleting: %s/%s", namespace, name)
			return nil
		}
		return err
	}
	klog.Info("Crontab try to process: %#v ..", crontab)
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)

}
func (c *Controller) onUpdate(newobj, oldobj interface{}) {

	// 判断资源版本是否一致 ，如果一致，不更新
	old := oldobj.(*v1beta1.CronTab)
	newo := newobj.(*v1beta1.CronTab)
	if old.ResourceVersion == newo.ResourceVersion {
		return
	}
	c.onAdd(newo)

}
func (c *Controller) onDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.onAdd(key)
}
