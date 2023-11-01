package components

import (
	"context"
	"crypto/sha1"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func HashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func FindSecret(client runtimeClient.Client, name, ns string) (*corev1.Secret, error) {
	secretObjectKey := runtimeClient.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	secret := &corev1.Secret{}
	err := client.Get(context.Background(), secretObjectKey, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
