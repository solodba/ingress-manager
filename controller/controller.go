package controller

import (
	"reflect"

	v12 "k8s.io/api/networking/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
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

// workqueue加入队列的方法
func (c *controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

// 添加Service方法
func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

// 更新Service方法
func (c *controller) updateService(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

// 删除Ingress方法
func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v12.Ingress)
	ownerReference := v13.GetControllerOf(ingress)
	if ownerReference == nil {
		return
	}
	if ownerReference.Kind != "Service" {
		return
	}
	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

// controller启动方法
func (c *controller) Run(stopCh chan struct{}) {
	<-stopCh
}
