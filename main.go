package main

import (
	"log"

	"github.com/solodba/ingress-manager/controller"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 读取k8s kubeconfig配置文件生成client-go的配置
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("cant not found k8s config")
		}
		config = inClusterConfig
	}

	// 通过配置生成k8s客户端
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	// 生成informer
	factory := informers.NewSharedInformerFactory(clientSet, 0)
	serviceInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	// 创建controller
	controller := controller.NewController(clientSet, serviceInformer, ingressInformer)

	// 启动informer
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	// 启动controller
	controller.Run(stopCh)
}
