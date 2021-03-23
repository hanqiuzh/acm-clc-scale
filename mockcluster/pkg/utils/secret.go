package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenHubClient(hubCfgPath string) (client.Client, error) {
	hubconfig, err := clientcmd.BuildConfigFromFlags("", hubCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to construct hubconfig %w", err)
	}

	return client.New(hubconfig, client.Options{})
}

// GenSaConfig will return a rest.Config which use the service account
// for the managed cluster.
// the generated cluster skips the TLS cert
func genSaClient(hClt client.Client, masterUrl, cluster string) (dynamic.Interface, error) {
	if hClt == nil {
		return nil, fmt.Errorf("got an invalid hub client")
	}
	ctx := context.TODO()

	saName := fmt.Sprintf("%s-bootstrap-sa ", cluster)
	saKey := types.NamespacedName{Name: saName, Namespace: cluster}

	sa := &corev1.ServiceAccount{}
	if err := hClt.Get(ctx, saKey, sa); err != nil {
		return nil, fmt.Errorf("failed to get the service account %w", err)
	}

	if len(sa.Secrets) == 0 {
		return nil, fmt.Errorf("service account doesn't have sercret attached")
	}

	secretPrefix := fmt.Sprintf("%s-token", saName)

	srtKey := types.NamespacedName{Namespace: cluster}
	for _, s := range sa.Secrets {
		if strings.HasPrefix(s.Name, secretPrefix) {
			srtKey.Name = s.Name
			break
		}
	}

	srt := &corev1.Secret{}

	if err := hClt.Get(ctx, srtKey, srt); err != nil {
		return nil, fmt.Errorf("failed to get secret attached to service account, err: %w", err)
	}

	tlsClientConfig := rest.TLSClientConfig{
		CertData: srt.Data["ca.crt"],
		Insecure: true,
	}

	saConfig := &rest.Config{
		// TODO: switch to using cluster DNS.
		Host:            masterUrl,
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(srt.Data["token"]),
	}

	// creates the clientset
	out, err := dynamic.NewForConfig(saConfig)
	if err != nil {
		return out, fmt.Errorf("failed to create mock spoke client, err %w", err)
	}

	return out, nil
}

func WiatForSpokeSa(hClt client.Client, masterUrl, cluster string) (dynamic.Interface, error) {
	tk := time.NewTicker(time.Second * 5)
	timeOut := time.After(time.Second * 90)

	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			sClt, err := genSaClient(hClt, masterUrl, cluster)
			if err == nil {
				return sClt, nil
			}
		case <-timeOut:
			return nil, fmt.Errorf("timeout: fail to generate spoke client")
		}

	}
}
