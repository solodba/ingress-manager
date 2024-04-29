package controller

import (
	serviceInformer "k8s.io/client-go/informers/core/v1"
	ingressInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	serviceLister "k8s.io/client-go/listers/core/v1"
	ingressLister "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// controller结构体
type controller struct {
	clientSet     kubernetes.Interface
	serviceLister serviceLister.ServiceLister
	ingressLister ingressLister.IngressLister
	queue         workqueue.RateLimitingInterface
}

// controller结构体构造函数
func NewController(clientSet kubernetes.Interface, serviceInformer serviceInformer.ServiceInformer, ingressInformer ingressInformer.IngressInformer) *controller {
	c := &controller{
		clientSet:     clientSet,
		serviceLister: serviceInformer.Lister(),
		ingressLister: ingressInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workqueue"),
	}
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
	})
	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})
	return c
}

// 添加Service方法
func (c *controller) addService(obj interface{}) {

}

// 更新Service方法
func (c *controller) updateService(oldObj, newObj interface{}) {

}

// 删除Ingress方法
func (c *controller) deleteIngress(obj interface{}) {

}

// controller启动方法
func (c *controller) Run(stopCh chan struct{}) {
	<-stopCh
}