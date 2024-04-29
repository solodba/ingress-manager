package controller

import (
	"context"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	serviceInformer "k8s.io/client-go/informers/core/v1"
	ingressInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	serviceLister "k8s.io/client-go/listers/core/v1"
	ingressLister "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	maxWorkNum  = 5
	maxRetryNum = 10
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
	for i := 0; i < maxWorkNum; i++ {
		go wait.Until(c.work, time.Minute, stopCh)
	}
	<-stopCh
}

// controller的工作处理函数
func (c *controller) work() {
	for c.processNextItem() {
	}
}

// controller的处理workqueue队列函数
func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	key := item.(string)
	err := c.syncService(key)
	if err != nil {
		c.handlerError(key, err)
	}
	return true
}

// controller的同步service操作函数
func (c *controller) syncService(key string) error {
	// 通过key获取service的namespace和name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// 查询是否存在Service
	service, err := c.clientSet.CoreV1().Services(namespace).Get(context.TODO(), name, v13.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// 查看Service是否存在annotations标签"ingress/http", Ingress是否存在
	_, ok := service.GetAnnotations()["ingress/http"]
	ingress, err := c.clientSet.NetworkingV1().Ingresses(namespace).Get(context.TODO(), name, v13.GetOptions{})
	if ok && errors.IsNotFound(err) {
		// 创建Ingress
		ing := c.serviceToIngress(service)
		_, err = c.clientSet.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ing, v13.CreateOptions{})
		if err != nil {
			return err
		}
	}
	if !ok && ingress != nil {
		// 删除Ingress
		err = c.clientSet.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, v13.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// 通过Service构造Ingress函数
func (c *controller) serviceToIngress(service *v1.Service) *v12.Ingress {
	ingress := &v12.Ingress{}
	ingress.Kind = "Ingress"
	ingress.APIVersion = "networking.k8s.io/v1"
	ingress.OwnerReferences = []v13.OwnerReference{
		*v13.NewControllerRef(service, v13.SchemeGroupVersion.WithKind("Service")),
	}
	ingress.Namespace = service.Namespace
	ingress.Name = service.Name
	ingressClassName := "nginx"
	pathType := v12.PathTypePrefix
	ingress.Spec = v12.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []v12.IngressRule{
			{
				Host: "codehorse.com",
				IngressRuleValue: v12.IngressRuleValue{
					HTTP: &v12.HTTPIngressRuleValue{
						Paths: []v12.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: v12.IngressBackend{
									Service: &v12.IngressServiceBackend{
										Name: service.Name,
										Port: v12.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return ingress
}

// controller的错误处理函数
func (c *controller) handlerError(key string, err error) {
	if c.queue.NumRequeues(key) <= maxRetryNum {
		c.queue.AddRateLimited(key)
	}
	runtime.HandleError(err)
	c.queue.Forget(key)
}
