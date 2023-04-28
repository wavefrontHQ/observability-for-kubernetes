package testhelper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockKubernetesDiscoveryClient struct {
	apiGroupList *metav1.APIGroupList
}

func NewMockKubernetesDiscoveryClient(availableResourceGroups []string) *MockKubernetesDiscoveryClient {
	apiGroupList := &metav1.APIGroupList{}
	for _, group := range availableResourceGroups {
		apiGroupList.Groups = append(apiGroupList.Groups, metav1.APIGroup{
			Name: group,
		})
	}

	return &MockKubernetesDiscoveryClient{
		apiGroupList: apiGroupList,
	}
}

func (kdc *MockKubernetesDiscoveryClient) ServerGroups() (*metav1.APIGroupList, error) {
	return kdc.apiGroupList, nil
}
