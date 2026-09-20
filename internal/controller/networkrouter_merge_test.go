// SPDX-License-Identifier: BSD-3-Clause

package controller

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/testify/v2/require"
	corev1 "k8s.io/api/core/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// basePT mirrors the operator's router base env: a valueFrom secret ref mixed
// with plain value vars, the same shape as production.
func basePT() *corev1ac.PodTemplateSpecApplyConfiguration {
	return corev1ac.PodTemplateSpec().WithSpec(corev1ac.PodSpec().WithContainers(
		corev1ac.Container().WithName("netbird").WithEnv(
			corev1ac.EnvVar().WithName("NB_SETUP_KEY").
				WithValueFrom(corev1ac.EnvVarSource().
					WithSecretKeyRef(corev1ac.SecretKeySelector().WithName("setup-key").WithKey("setup-key"))),
			corev1ac.EnvVar().WithName("NB_MANAGEMENT_URL").WithValue("https://api.netbird.io"),
			corev1ac.EnvVar().WithName("NB_LOG_LEVEL").WithValue("info"),
		),
	))
}

func TestWorkloadOverrideEnvMerge(t *testing.T) {
	t.Parallel()

	baseJSON, err := json.Marshal(basePT())
	require.NoError(t, err)
	override := []byte(`{"spec":{"containers":[{"name":"netbird","env":[` +
		`{"name":"NB_ENABLE_ROSENPASS","value":"true"},` +
		`{"name":"NB_ROSENPASS_PERMISSIVE","value":"true"}]}` +
		`]}}`)
	merged, err := strategicpatch.StrategicMergePatch(baseJSON, override, corev1.PodTemplateSpec{})
	require.NoError(t, err)

	// The fix: unmarshal the merged result into a fresh builder.
	fresh := &corev1ac.PodTemplateSpecApplyConfiguration{}
	require.NoError(t, json.Unmarshal(merged, fresh))

	// No env var may carry both value and valueFrom (the API server rejects it).
	for _, c := range fresh.Spec.Containers {
		for _, e := range c.Env {
			require.True(t, e.Value == nil || e.ValueFrom == nil,
				"env %s has both value and valueFrom", *e.Name)
		}
	}
}
