// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kubelet

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	util "k8s.io/client-go/util/testing"
)

func TestGetPodList(t *testing.T) {
	t.Run("it returns a list of Pods", func(t *testing.T) {
		podsContent, err := ioutil.ReadFile("k8s_api_pods.json")
		require.NoError(t, err)

		ip, port, closeServer := setupTestServer(t, "/pods", string(podsContent))
		defer closeServer()

		kubeletClientConfig := &KubeletClientConfig{
			Port: port,
		}
		kubeletClient, err := NewKubeletClient(kubeletClientConfig)
		require.NoError(t, err)

		pods, err := kubeletClient.GetPodList(ip)
		require.NoError(t, err)

		require.Len(t, pods.Items, 7)
		assert.Equal(t, pods.Items[0].Name, "mult-job-success-8cpzn", "Expected first Pod name to be `mult-job-success-8cpzn`")
	})

	t.Run("it returns an error if the request to the kublet server fails", func(t *testing.T) {
		kubeletClientConfig := &KubeletClientConfig{
			Port: 12345,
		}
		kubeletClient, err := NewKubeletClient(kubeletClientConfig)
		require.NoError(t, err)

		_, err = kubeletClient.GetPodList(nil)
		require.Error(t, err)
	})
}

func TestGetSummary(t *testing.T) {
	t.Run("it returns a summary", func(t *testing.T) {
		summaryContent, err := ioutil.ReadFile("k8s_api_summary.json")
		require.NoError(t, err)

		ip, port, closeServer := setupTestServer(t, "/stats/summary", string(summaryContent))
		defer closeServer()

		kubeletClientConfig := &KubeletClientConfig{
			Port: port,
		}
		kubeletClient, err := NewKubeletClient(kubeletClientConfig)
		require.NoError(t, err)

		summary, err := kubeletClient.GetSummary(ip)
		require.NoError(t, err)

		require.Len(t, summary.Pods, 2)
		assert.Equal(t, summary.Node.NodeName, "some-node")
		assert.Equal(t, summary.Pods[0].PodRef.Name, "some-pod")
	})

	t.Run("it returns an error if the request to the kublet server fails", func(t *testing.T) {
		kubeletClientConfig := &KubeletClientConfig{
			Port: 12345,
		}
		kubeletClient, err := NewKubeletClient(kubeletClientConfig)
		require.NoError(t, err)

		_, err = kubeletClient.GetSummary(nil)
		require.Error(t, err)
	})
}

func setupTestServer(t *testing.T, endpoint string, content string) (net.IP, uint, func()) {
	podsHandler := util.FakeHandler{
		StatusCode:   http.StatusOK,
		RequestBody:  "",
		ResponseBody: content,
		T:            t,
	}

	router := http.NewServeMux()
	router.Handle(endpoint, &podsHandler)

	server := httptest.NewServer(router)
	mockServerUrl, _ := url.Parse(server.URL)
	_, port, _ := net.SplitHostPort(mockServerUrl.Host)
	mockPort, _ := strconv.ParseUint(port, 10, 64)
	return net.ParseIP("127.0.0.1"), uint(mockPort), server.Close
}
