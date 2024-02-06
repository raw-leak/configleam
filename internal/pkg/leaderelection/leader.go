package leaderelection

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	DefaultLeaseDuration = 15 * time.Second
	DefaultRenewDeadline = 10 * time.Second
	DefaultRetryPeriod   = 2 * time.Second
)

// LeaderElectionConfig represents the configuration for leader election
type LeaderElectionConfig struct {
	LeaseLockName      string        // Name of the lease lock
	LeaseLockNamespace string        // Namespace for the lease lock
	Identity           string        // Unique identity for the candidate
	LeaseDuration      time.Duration // Duration that non-leader candidates will wait to force acquire leadership
	RenewDeadline      time.Duration // Duration that the acting master will retry refreshing leadership before giving up
	RetryPeriod        time.Duration // Duration the LeaderElector clients should wait between tries of actions
}

// New creates a new leader election manager
func New(cfg *LeaderElectionConfig, onStart, onStop func()) (*leaderelection.LeaderElector, error) {
	// set default values
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = DefaultLeaseDuration
	}
	if cfg.RenewDeadline == 0 {
		cfg.RenewDeadline = DefaultRenewDeadline
	}
	if cfg.RetryPeriod == 0 {
		cfg.RetryPeriod = DefaultRetryPeriod
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.LeaseLockName,
			Namespace: cfg.LeaseLockNamespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.Identity,
		},
	}

	leCfg := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: cfg.LeaseDuration,
		RenewDeadline: cfg.RenewDeadline,
		RetryPeriod:   cfg.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				onStart()
			},
			OnStoppedLeading: func() {
				onStop()
			},
		},
	}

	elector, err := leaderelection.NewLeaderElector(leCfg)
	if err != nil {
		return nil, err
	}

	return elector, nil
}

// Run starts the leader election process
func Run(elector *leaderelection.LeaderElector, ctx context.Context) {

	elector.Run(ctx)

	<-ctx.Done()
	runtime.HandleCrash()
}
