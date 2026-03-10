package sandbox

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sandboxv1alpha1 "sigs.k8s.io/agent-sandbox/api/v1alpha1"
	sandboxclient "sigs.k8s.io/agent-sandbox/clients/k8s/clientset/versioned"
)

const (
	lockPrefix = "lock:ensure:"
	lockTTL    = 3 * time.Second
)

type Config struct {
	Namespace    string // K8s namespace for sandboxes
	Image        string // Agent container image
	RedisAddr    string // Passed to sandbox as env var
	StreamPrefix string // Passed to sandbox as env var
}

type Orchestrator struct {
	client sandboxclient.Interface
	redis  *redis.Client
	cfg    Config
}

func NewOrchestrator(client sandboxclient.Interface, rdb *redis.Client, cfg Config) *Orchestrator {
	return &Orchestrator{
		client: client,
		redis:  rdb,
		cfg:    cfg,
	}
}

// Ensure creates or confirms the Sandbox CR for a session.
// Uses a Redis lock to prevent ensure storms. All errors are logged, never returned.
func (o *Orchestrator) Ensure(ctx context.Context, sessionKey, tenantID, chatType string) {
	locked, err := o.redis.SetNX(ctx, lockPrefix+sessionKey, "1", lockTTL).Result()
	if err != nil {
		log.Printf("ensure lock check failed: session_key=%s err=%v", sessionKey, err)
		return
	}
	if !locked {
		return
	}

	name := sandboxName(sessionKey)
	sbx := buildSandbox(name, o.cfg, sessionKey, tenantID, chatType)

	_, err = o.client.AgentsV1alpha1().Sandboxes(o.cfg.Namespace).Create(ctx, sbx, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return
	}
	if err != nil {
		log.Printf("ensure sandbox create failed: session_key=%s sandbox=%s err=%v", sessionKey, name, err)
		return
	}
	log.Printf("ensure sandbox created: session_key=%s sandbox=%s", sessionKey, name)
}

func buildSandbox(name string, cfg Config, sessionKey, tenantID, chatType string) *sandboxv1alpha1.Sandbox {
	return &sandboxv1alpha1.Sandbox{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "tinyclaw-sandbox",
			},
		},
		Spec: sandboxv1alpha1.SandboxSpec{
			PodTemplate: sandboxv1alpha1.PodTemplate{
				ObjectMeta: sandboxv1alpha1.PodMetadata{
					Labels: map[string]string{
						"tinyclaw/session-key": sessionKey,
						"tinyclaw/tenant-id":   tenantID,
						"tinyclaw/chat-type":   chatType,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: cfg.Image,
							Env: []corev1.EnvVar{
								{Name: "SESSION_KEY", Value: sessionKey},
								{Name: "REDIS_ADDR", Value: cfg.RedisAddr},
								{Name: "STREAM_PREFIX", Value: cfg.StreamPrefix},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}

// sandboxName returns a deterministic Sandbox name for a session key.
func sandboxName(sessionKey string) string {
	return "tinyclaw-agent-" + sessionKey
}
