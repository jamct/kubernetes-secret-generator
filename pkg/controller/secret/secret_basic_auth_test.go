package secret

import (
	"context"
	"github.com/imdario/mergo"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

func newBasicAuthTestSecret(extraAnnotations map[string]string) *corev1.Secret {
	annotations := map[string]string{
		AnnotationSecretType: string(SecretTypeBasicAuth),
	}

	if extraAnnotations != nil {
		if err := mergo.Merge(&annotations, extraAnnotations, mergo.WithOverride); err != nil {
			panic(err)
		}
	}

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName(),
			Namespace: "default",
			Labels: map[string]string{
				labelSecretGeneratorTest: "yes",
			},
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	return s
}

// verify basic fields of the secret are present
func verifyBasicAuthSecret(t *testing.T, in, out *corev1.Secret) {
	if out.Annotations[AnnotationSecretType] != string(SecretTypeBasicAuth) {
		t.Errorf("generated secret has wrong type %s on  %s annotation", out.Annotations[AnnotationSecretType], AnnotationSecretType)
	}

	_, wasGenerated := in.Annotations[AnnotationSecretAutoGeneratedAt]

	auth := out.Data[SecretFieldBasicAuthIngress]
	password := out.Data[SecretFieldBasicAuthPassword]

	// check if password has been saved in clear text
	// and has correct length (if the secret has actually been generated)
	if !wasGenerated && (len(password) == 0 || len(password) != desiredLength(in)) {
		t.Errorf("generated field has wrong length of %d", len(password))
	}

	// check if auth field has been generated (with separator)
	if len(auth) == 0 || !strings.Contains(string(auth), ":") {
		t.Errorf("auth field has wrong or no values %s", string(auth))
	}

	if _, ok := out.Annotations[AnnotationSecretAutoGeneratedAt]; !ok {
		t.Errorf("secret has no %s annotation", AnnotationSecretAutoGeneratedAt)
	}
}

func TestGenerateBasicAuthWithoutUsername(t *testing.T) {
	in := newBasicAuthTestSecret(map[string]string{})
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyBasicAuthSecret(t, in, out)
	require.Equal(t, "admin", string(out.Data[SecretFieldBasicAuthUsername]))
}

func TestGenerateBasicAuthWithUsername(t *testing.T) {
	in := newBasicAuthTestSecret(map[string]string{
		AnnotationBasicAuthUsername: "test123",
	})
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyBasicAuthSecret(t, in, out)
	require.Equal(t, "test123", string(out.Data[SecretFieldBasicAuthUsername]))
}

func TestGenerateBasicAuthRegenerate(t *testing.T) {
	in := newBasicAuthTestSecret(map[string]string{
		AnnotationBasicAuthUsername: "test123",
	})
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyBasicAuthSecret(t, in, out)
	require.Equal(t, "test123", string(out.Data[SecretFieldBasicAuthUsername]))
	oldPassword := string(out.Data[SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[SecretFieldBasicAuthIngress])

	// force regenerate
	out.Annotations[AnnotationSecretRegenerate] = "yes"
	require.NoError(t, mgr.GetClient().Update(context.TODO(), out))

	doReconcile(t, out, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))
	newPassword := string(outNew.Data[SecretFieldBasicAuthPassword])
	newAuth := string(outNew.Data[SecretFieldBasicAuthIngress])

	if oldPassword == newPassword {
		t.Errorf("secret has not been updated")
	}

	if oldAuth == newAuth {
		t.Errorf("secret has not been updated")
	}
}

func TestGenerateBasicAuthNoRegenerate(t *testing.T) {
	in := newBasicAuthTestSecret(map[string]string{
		AnnotationBasicAuthUsername: "test123",
	})
	require.NoError(t, mgr.GetClient().Create(context.TODO(), in))

	doReconcile(t, in, false)

	out := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, out))

	verifyBasicAuthSecret(t, in, out)
	require.Equal(t, "test123", string(out.Data[SecretFieldBasicAuthUsername]))
	oldPassword := string(out.Data[SecretFieldBasicAuthPassword])
	oldAuth := string(out.Data[SecretFieldBasicAuthIngress])

	doReconcile(t, in, false)

	outNew := &corev1.Secret{}
	require.NoError(t, mgr.GetClient().Get(context.TODO(), client.ObjectKey{
		Name:      in.Name,
		Namespace: in.Namespace}, outNew))
	newPassword := string(out.Data[SecretFieldBasicAuthPassword])
	newAuth := string(out.Data[SecretFieldBasicAuthIngress])

	if oldPassword != newPassword {
		t.Errorf("secret has been updated")
	}

	if oldAuth != newAuth {
		t.Errorf("secret has been updated")
	}
}
