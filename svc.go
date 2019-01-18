package inClusterServiceDiscovery

import (
	"fmt"
	"log"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//v1beta1 "k8s.io/api/extensions/v1beta1"
)

type Search struct {
	ControllerName string
	ControllerType string
	Namespace      string
	Service        string
	Separator      string
	Try            int
	Total          int
	Client         *kubernetes.Clientset
	Ip             string
}

func CreateSearch() (*Search, error) {
	s := Search{}
	s.Namespace = "default"
	s.Try = 100
	s.Separator = ","
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	s.Client = cli
	return &s, err
}

func (this *Search) Result() (string, error) {
	var ret int
	var err error
	switch this.CntrollerType {
	case "daemonset", "ds":
		ret, err = this.Daemonset()
	case "deployment", "deploy":
		ret, err = this.Deployment()
	case "statefulset", "state", "s":
		ret, err = this.Statefulset()
	default:
		err = fmt.Sprintf("err: wrong type of controller, as instance: deployment, statefulset or daemonset")
		ret = nil
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}
	this.Total = ret
	ip, err := this.GetEndpoints()
	return ip, err
}

func (this *Search) Daemonset() (int, error) {
	cli := this.Client
	namespace := this.Namespace
	name := this.ControllerName
	svc := this.Service
	obj, err := cli.ExtensionsV1beta1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	total := int(obj.Status.DesiredNumberScheduled)
	return total, err
}

func (this *Search) Deployment() (string, error) {
	cli := this.Client
	namespace := this.Namespace
	name := this.ControllerName
	svc := this.Service
	obj, err := cli.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	total := int(*(obj.Spec.Replicas)) // deployment, statefulset
	return total, err
}

func (this *Search) Statefulset() (string, nil) {
	cli := this.Client
	namespace := this.Namespace
	name := this.ControllerName
	svc := this.Service
	obj, err := cli.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	total := int(*(obj.Spec.Replicas)) // deployment, statefulset
	return total, err
}

func (this *Search) GetEndpoints() (string, error) {
	cli := this.Client
	namespace := this.Namespace
	svc := this.Service
	for try := 0; try < this.Try; try++ {
		eps, err := cli.CoreV1().Endpoints(namespace).Get(svc, metav1.GetOptions{})
		if err != nil {
			log.Println(err)
			return nil, err
		}
		n1 := len(eps.Subsets)
		for i := 0; i < n1; i++ {
			addrs := eps.Subsets[i].Addresses
			n2 := len(addrs)
			if n2 == this.Total {
				ips := ""
				sep := ""
				for j := 0; j < n2; j++ {
					ips += sep
					ips += fmt.Sprintf("%v", addrs[j].IP)
					sep = this.Separator
				}
				return ips, err
			}
			time.Sleep(3 * time.Second)
		}
	}
	err = fmt.Sprintf("cannot find IP of %v.%v", this.Service, this.Namespace)
	log.Println(err)
	return nil, err
}