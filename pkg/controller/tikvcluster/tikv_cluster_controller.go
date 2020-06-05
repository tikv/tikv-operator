// Copyright 2018 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package tikvcluster

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/client/clientset/versioned"
	informers "github.com/tikv/tikv-operator/pkg/client/informers/externalversions"
	listers "github.com/tikv/tikv-operator/pkg/client/listers/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/controller"
	mm "github.com/tikv/tikv-operator/pkg/manager/member"
	"github.com/tikv/tikv-operator/pkg/manager/meta"
	"github.com/tikv/tikv-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Controller controls tikvclusters.
type Controller struct {
	// kubernetes client interface
	kubeClient kubernetes.Interface
	// operator client interface
	cli versioned.Interface
	// control returns an interface capable of syncing a tikv cluster.
	// Abstracted out for testing.
	control ControlInterface
	// tcLister is able to list/get tikvclusters from a shared informer's store
	tcLister listers.TikvClusterLister
	// tcListerSynced returns true if the tikvcluster shared informer has synced at least once
	tcListerSynced cache.InformerSynced
	// setLister is able to list/get stateful sets from a shared informer's store
	setLister appslisters.StatefulSetLister
	// setListerSynced returns true if the statefulset shared informer has synced at least once
	setListerSynced cache.InformerSynced
	// tikvclusters that need to be synced.
	queue workqueue.RateLimitingInterface
}

// NewController creates a tikvcluster controller.
func NewController(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	autoFailover bool,
	pdFailoverPeriod time.Duration,
	tikvFailoverPeriod time.Duration,
) *Controller {
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{QPS: 1})
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tikv-controller-manager"})

	tcInformer := informerFactory.Tikv().V1alpha1().TikvClusters()
	setInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	svcInformer := kubeInformerFactory.Core().V1().Services()
	epsInformer := kubeInformerFactory.Core().V1().Endpoints()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	tcControl := controller.NewRealTikvClusterControl(cli, tcInformer.Lister(), recorder)
	pdControl := pdapi.NewDefaultPDControl(kubeCli)
	setControl := controller.NewRealStatefuSetControl(kubeCli, setInformer.Lister(), recorder)
	svcControl := controller.NewRealServiceControl(kubeCli, svcInformer.Lister(), recorder)
	pvControl := controller.NewRealPVControl(kubeCli, pvcInformer.Lister(), pvInformer.Lister(), recorder)
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())
	podControl := controller.NewRealPodControl(kubeCli, pdControl, podInformer.Lister(), recorder)
	typedControl := controller.NewTypedControl(controller.NewRealGenericControl(genericCli, recorder))
	pdScaler := mm.NewPDScaler(pdControl, pvcInformer.Lister(), pvcControl)
	tikvScaler := mm.NewTiKVScaler(pdControl, pvcInformer.Lister(), pvcControl, podInformer.Lister())
	pdFailover := mm.NewPDFailover(cli, pdControl, pdFailoverPeriod, podInformer.Lister(), podControl, pvcInformer.Lister(), pvcControl, pvInformer.Lister(), recorder)
	tikvFailover := mm.NewTiKVFailover(tikvFailoverPeriod, recorder)
	pdUpgrader := mm.NewPDUpgrader(pdControl, podControl, podInformer.Lister())
	tikvUpgrader := mm.NewTiKVUpgrader(pdControl, podControl, podInformer.Lister())

	tcc := &Controller{
		kubeClient: kubeCli,
		cli:        cli,
		control: NewDefaultTikvClusterControl(
			tcControl,
			mm.NewPDMemberManager(
				pdControl,
				setControl,
				svcControl,
				podControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				epsInformer.Lister(),
				pvcInformer.Lister(),
				pdScaler,
				pdUpgrader,
				autoFailover,
				pdFailover,
			),
			mm.NewTiKVMemberManager(
				pdControl,
				setControl,
				svcControl,
				typedControl,
				setInformer.Lister(),
				svcInformer.Lister(),
				podInformer.Lister(),
				nodeInformer.Lister(),
				autoFailover,
				tikvFailover,
				tikvScaler,
				tikvUpgrader,
			),
			meta.NewMetaManager(
				pvcInformer.Lister(),
				pvcControl,
				pvInformer.Lister(),
				pvControl,
				podInformer.Lister(),
				podControl,
			),
			mm.NewOrphanPodsCleaner(
				podInformer.Lister(),
				podControl,
				pvcInformer.Lister(),
				kubeCli,
			),
			mm.NewPDDiscoveryManager(typedControl),
			&tikvClusterConditionUpdater{},
			recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"tikvcluster",
		),
	}

	tcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: tcc.enqueueTikvCluster,
		UpdateFunc: func(old, cur interface{}) {
			tcc.enqueueTikvCluster(cur)
		},
		DeleteFunc: tcc.enqueueTikvCluster,
	})
	tcc.tcLister = tcInformer.Lister()
	tcc.tcListerSynced = tcInformer.Informer().HasSynced

	setInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: tcc.addStatefulSet,
		UpdateFunc: func(old, cur interface{}) {
			tcc.updateStatefuSet(old, cur)
		},
		DeleteFunc: tcc.deleteStatefulSet,
	})
	tcc.setLister = setInformer.Lister()
	tcc.setListerSynced = setInformer.Informer().HasSynced

	return tcc
}

// Run runs the tikvcluster controller.
func (tcc *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer tcc.queue.ShutDown()

	klog.Info("Starting tikvcluster controller")
	defer klog.Info("Shutting down tikvcluster controller")

	for i := 0; i < workers; i++ {
		go wait.Until(tcc.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (tcc *Controller) worker() {
	for tcc.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (tcc *Controller) processNextWorkItem() bool {
	key, quit := tcc.queue.Get()
	if quit {
		return false
	}
	defer tcc.queue.Done(key)
	if err := tcc.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("TikvCluster: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("TikvCluster: %v, sync failed %v, requeuing", key.(string), err))
		}
		tcc.queue.AddRateLimited(key)
	} else {
		tcc.queue.Forget(key)
	}
	return true
}

// sync syncs the given tikvcluster.
func (tcc *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing TikvCluster %q (%v)", key, time.Since(startTime))
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	tc, err := tcc.tcLister.TikvClusters(ns).Get(name)
	if errors.IsNotFound(err) {
		klog.Infof("TikvCluster has been deleted %v", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tcc.syncTikvCluster(tc.DeepCopy())
}

func (tcc *Controller) syncTikvCluster(tc *v1alpha1.TikvCluster) error {
	return tcc.control.UpdateTikvCluster(tc)
}

// enqueueTikvCluster enqueues the given tikvcluster in the work queue.
func (tcc *Controller) enqueueTikvCluster(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	tcc.queue.Add(key)
}

// addStatefulSet adds the tikvcluster for the statefulset to the sync queue
func (tcc *Controller) addStatefulSet(obj interface{}) {
	set := obj.(*apps.StatefulSet)
	ns := set.GetNamespace()
	setName := set.GetName()

	if set.DeletionTimestamp != nil {
		// on a restart of the controller manager, it's possible a new statefulset shows up in a state that
		// is already pending deletion. Prevent the statefulset from being a creation observation.
		tcc.deleteStatefulSet(set)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	tc := tcc.resolveTikvClusterFromSet(ns, set)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefuSet %s/%s created, TikvCluster: %s/%s", ns, setName, ns, tc.Name)
	tcc.enqueueTikvCluster(tc)
}

// updateStatefuSet adds the tikvcluster for the current and old statefulsets to the sync queue.
func (tcc *Controller) updateStatefuSet(old, cur interface{}) {
	curSet := cur.(*apps.StatefulSet)
	oldSet := old.(*apps.StatefulSet)
	ns := curSet.GetNamespace()
	setName := curSet.GetName()
	if curSet.ResourceVersion == oldSet.ResourceVersion {
		// Periodic resync will send update events for all known statefulsets.
		// Two different versions of the same statefulset will always have different RVs.
		return
	}

	// If it has a ControllerRef, that's all that matters.
	tc := tcc.resolveTikvClusterFromSet(ns, curSet)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s updated, %+v -> %+v.", ns, setName, oldSet.Spec, curSet.Spec)
	tcc.enqueueTikvCluster(tc)
}

// deleteStatefulSet enqueues the tikvcluster for the statefulset accounting for deletion tombstones.
func (tcc *Controller) deleteStatefulSet(obj interface{}) {
	set, ok := obj.(*apps.StatefulSet)
	ns := set.GetNamespace()
	setName := set.GetName()

	// When a delete is dropped, the relist will notice a statefuset in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %+v", obj))
			return
		}
		set, ok = tombstone.Obj.(*apps.StatefulSet)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a statefuset %+v", obj))
			return
		}
	}

	// If it has a TikvCluster, that's all that matters.
	tc := tcc.resolveTikvClusterFromSet(ns, set)
	if tc == nil {
		return
	}
	klog.V(4).Infof("StatefulSet %s/%s deleted through %v.", ns, setName, utilruntime.GetCaller())
	tcc.enqueueTikvCluster(tc)
}

// resolveTikvClusterFromSet returns the TikvCluster by a StatefulSet,
// or nil if the StatefulSet could not be resolved to a matching TikvCluster
// of the correct Kind.
func (tcc *Controller) resolveTikvClusterFromSet(namespace string, set *apps.StatefulSet) *v1alpha1.TikvCluster {
	controllerRef := metav1.GetControllerOf(set)
	if controllerRef == nil {
		return nil
	}

	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controller.ControllerKind.Kind {
		return nil
	}
	tc, err := tcc.tcLister.TikvClusters(namespace).Get(controllerRef.Name)
	if err != nil {
		return nil
	}
	if tc.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return tc
}
