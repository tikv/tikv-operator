// Copyright 2020 TiKV Project Authors.
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

package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/tikv/tikv-operator/pkg/client/clientset/versioned"
	informers "github.com/tikv/tikv-operator/pkg/client/informers/externalversions"
	"github.com/tikv/tikv-operator/pkg/controller"
	"github.com/tikv/tikv-operator/pkg/controller/tikvcluster"
	"github.com/tikv/tikv-operator/pkg/scheme"
	"github.com/tikv/tikv-operator/pkg/verflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/util/term"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/version"
	"k8s.io/klog"
	utilflag "k8s.io/kubernetes/pkg/util/flag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	workers            int
	autoFailover       bool
	pdFailoverPeriod   time.Duration
	tikvFailoverPeriod time.Duration
	leaseDuration      = 15 * time.Second
	renewDuration      = 5 * time.Second
	retryPeriod        = 3 * time.Second
	waitDuration       = 5 * time.Second
	namedFlagSets      cliflag.NamedFlagSets
)

// TODO organize via component config/option
func initFlags(fs *flag.FlagSet) {
	fs.IntVar(&workers, "workers", 5, "The number of workers that are allowed to sync concurrently. Larger number = more responsive management, but more CPU (and network) load")
	fs.BoolVar(&autoFailover, "auto-failover", true, "Auto failover")
	fs.DurationVar(&pdFailoverPeriod, "pd-failover-period", time.Duration(5*time.Minute), "PD failover period default(5m)")
	fs.DurationVar(&tikvFailoverPeriod, "tikv-failover-period", time.Duration(5*time.Minute), "TiKV failover period default(5m)")
	fs.DurationVar(&controller.ResyncDuration, "resync-duration", time.Duration(30*time.Second), "Resync time of informer")
	fs.StringVar(&controller.PDDiscoveryImage, "pd-discovery-image", "tikv/tikv-operator:latest", "The image of the PD discovery service")
}

// Run runs the controller-manager. This should never exit.
func Run(stopCh <-chan struct{}) error {
	hostName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("failed to get hostname: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		klog.Fatal("NAMESPACE environment variable not set")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create Clientset: %v", err)
	}
	var kubeCli kubernetes.Interface
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	genericCli, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		klog.Fatalf("failed to get the generic kube-apiserver client: %v", err)
	}

	var informerFactory informers.SharedInformerFactory
	var kubeInformerFactory kubeinformers.SharedInformerFactory
	var options []informers.SharedInformerOption
	var kubeoptions []kubeinformers.SharedInformerOption
	informerFactory = informers.NewSharedInformerFactoryWithOptions(cli, controller.ResyncDuration, options...)
	kubeInformerFactory = kubeinformers.NewSharedInformerFactoryWithOptions(kubeCli, controller.ResyncDuration, kubeoptions...)

	rl := resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "tikv-controller-manager",
		},
		Client: kubeCli.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      hostName,
			EventRecorder: &record.FakeRecorder{},
		},
	}

	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onStarted := func(ctx context.Context) {
		_ = genericCli
		tcController := tikvcluster.NewController(kubeCli, cli, genericCli, informerFactory, kubeInformerFactory, autoFailover, pdFailoverPeriod, tikvFailoverPeriod)

		// Start informer factories after all controller are initialized.
		informerFactory.Start(ctx.Done())
		kubeInformerFactory.Start(ctx.Done())

		// Wait for all started informers' cache were synced.
		for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
			if !synced {
				klog.Fatalf("error syncing informer for %v", v)
			}
		}
		for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
			if !synced {
				klog.Fatalf("error syncing informer for %v", v)
			}
		}
		klog.Infof("cache of informer factories sync successfully")

		wait.Forever(func() { tcController.Run(workers, ctx.Done()) }, waitDuration)
	}

	onStopped := func() {
		klog.Fatalf("leader election lost")
	}

	// leader election for multiple tikv-controller-manager instances
	go wait.Forever(func() {
		leaderelection.RunOrDie(controllerCtx, leaderelection.LeaderElectionConfig{
			Lock:          &rl,
			LeaseDuration: leaseDuration,
			RenewDeadline: renewDuration,
			RetryPeriod:   retryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: onStarted,
				OnStoppedLeading: onStopped,
			},
		})
	}, waitDuration)

	healthz.InstallHandler(http.DefaultServeMux)
	klog.Fatal(http.ListenAndServe(":6060", nil))
	return nil
}

func NewControllerManagerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `TiKV Controller Manager`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			klog.Infof("TiKV Controller Manager: %s", version.Get())
			utilflag.PrintFlags(flag.CommandLine)

			if err := Run(wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	initFlags(namedFlagSets.FlagSet("generic"))
	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
	for _, f := range namedFlagSets.FlagSets {
		flag.CommandLine.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	return cmd
}
