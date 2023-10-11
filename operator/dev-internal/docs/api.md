# API Reference

Packages:

- [wavefront.com/v1alpha1](#wavefrontcomv1alpha1)

# wavefront.com/v1alpha1

Resource Types:

- [Wavefront](#wavefront)




## Wavefront
<sup><sup>[↩ Parent](#wavefrontcomv1alpha1 )</sup></sup>






Wavefront is the Schema for the wavefronts API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>wavefront.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Wavefront</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#wavefrontspec">spec</a></b></td>
        <td>object</td>
        <td>
          WavefrontSpec defines the desired state of Wavefront<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontstatus">status</a></b></td>
        <td>object</td>
        <td>
          WavefrontStatus defines the observed state of Wavefront<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec
<sup><sup>[↩ Parent](#wavefront)</sup></sup>



WavefrontSpec defines the desired state of Wavefront

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clusterName</b></td>
        <td>string</td>
        <td>
          ClusterName is a unique name for the Kubernetes cluster to be identified via a metric tag on Wavefront (Required).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>allowLegacyInstall</b></td>
        <td>boolean</td>
        <td>
          Allows the operator based Wavefront installation to be run in parallel with a legacy Wavefront (helm or manual) installation. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollection">dataCollection</a></b></td>
        <td>object</td>
        <td>
          DataCollection options<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexport">dataExport</a></b></td>
        <td>object</td>
        <td>
          DataExport options<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimental">experimental</a></b></td>
        <td>object</td>
        <td>
          Experimental features<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullSecret</b></td>
        <td>string</td>
        <td>
          ImagePullSecret is the name of the secret to authenticate with a private custom registry.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>wavefrontTokenSecret</b></td>
        <td>string</td>
        <td>
          WavefrontTokenSecret is the name of the secret that contains a wavefront API Token.<br/>
          <br/>
            <i>Default</i>: wavefront-secret<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>wavefrontUrl</b></td>
        <td>string</td>
        <td>
          Wavefront URL for your cluster<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection
<sup><sup>[↩ Parent](#wavefrontspec)</sup></sup>



DataCollection options

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionlogging">logging</a></b></td>
        <td>object</td>
        <td>
          Enable and configure wavefront logging<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          Metrics has resource configuration for node- and cluster-deployed collectors<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectiontolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          Top level tolerations to be applied to metrics and logging DaemonSet resource types. This adds custom tolerations to the pods.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.logging
<sup><sup>[↩ Parent](#wavefrontspecdatacollection)</sup></sup>



Enable and configure wavefront logging

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable is whether to enable the wavefront logging. Defaults to false.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionloggingfilters">filters</a></b></td>
        <td>object</td>
        <td>
          Filters to apply towards all logs collected by wavefront-logging.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionloggingresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the logging containers.<br/>
          <br/>
            <i>Default</i>: map[limits:map[cpu:200m ephemeral-storage:2Gi memory:256Mi] requests:map[cpu:100m ephemeral-storage:1Gi memory:50Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>map[string]string</td>
        <td>
          Tags are a map of key value pairs that are added to all logging emitted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.logging.filters
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionlogging)</sup></sup>



Filters to apply towards all logs collected by wavefront-logging.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>tagAllowList</b></td>
        <td>map[string][]string</td>
        <td>
          List of log tag patterns to allow<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tagDenyList</b></td>
        <td>map[string][]string</td>
        <td>
          List of log tag patterns to deny<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.logging.resources
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionlogging)</sup></sup>



Resources Compute resources required by the logging containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionloggingresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionloggingresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.logging.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionloggingresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.logging.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionloggingresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics
<sup><sup>[↩ Parent](#wavefrontspecdatacollection)</sup></sup>



Metrics has resource configuration for node- and cluster-deployed collectors

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsclustercollector">clusterCollector</a></b></td>
        <td>object</td>
        <td>
          ClusterCollector is for resource configuration for the cluster collector.<br/>
          <br/>
            <i>Default</i>: map[resources:map[limits:map[cpu:2000m ephemeral-storage:1Gi memory:512Mi] requests:map[cpu:200m ephemeral-storage:20Mi memory:10Mi]]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricscontrolplane">controlPlane</a></b></td>
        <td>object</td>
        <td>
          Whether to enable or disable metrics from the Kubernetes control plane. Defaults to true.<br/>
          <br/>
            <i>Default</i>: map[enable:true]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>customConfig</b></td>
        <td>string</td>
        <td>
          CustomConfig is the custom ConfigMap name for the collector. Leave blank to use defaults.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>defaultCollectionInterval</b></td>
        <td>string</td>
        <td>
          Default metrics collection interval. Defaults to 60s.<br/>
          <br/>
            <i>Default</i>: 60s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable is whether to enable the metrics. Defaults to true.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableDiscovery</b></td>
        <td>boolean</td>
        <td>
          Rules based and Prometheus endpoints auto-discovery. Defaults to true.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsfilters">filters</a></b></td>
        <td>object</td>
        <td>
          Filters to apply towards all metrics collected by the collector.<br/>
          <br/>
            <i>Default</i>: map[denyList:[kubernetes.sys_container.* kubernetes.collector.runtime.* kubernetes.*.network.rx_rate kubernetes.*.network.rx_errors_rate kubernetes.*.network.tx_rate kubernetes.*.network.tx_errors_rate kubernetes.*.memory.page_faults kubernetes.*.memory.page_faults_rate kubernetes.*.memory.major_page_faults kubernetes.*.memory.major_page_faults_rate kubernetes.*.filesystem.inodes kubernetes.*.filesystem.inodes_free kubernetes.*.ephemeral_storage.request kubernetes.*.ephemeral_storage.limit]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsnodecollector">nodeCollector</a></b></td>
        <td>object</td>
        <td>
          NodeCollector is for resource configuration for the node collector.<br/>
          <br/>
            <i>Default</i>: map[resources:map[limits:map[cpu:1000m ephemeral-storage:512Mi memory:256Mi] requests:map[cpu:200m ephemeral-storage:20Mi memory:10Mi]]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tags</b></td>
        <td>map[string]string</td>
        <td>
          Tags are a map of key value pairs that are added as point tags on all metrics emitted.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.clusterCollector
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetrics)</sup></sup>



ClusterCollector is for resource configuration for the cluster collector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsclustercollectorresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the Collector containers.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.clusterCollector.resources
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsclustercollector)</sup></sup>



Resources Compute resources required by the Collector containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsclustercollectorresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsclustercollectorresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.clusterCollector.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsclustercollectorresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.clusterCollector.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsclustercollectorresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.controlPlane
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetrics)</sup></sup>



Whether to enable or disable metrics from the Kubernetes control plane. Defaults to true.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable is whether to include kubernetes.controlplane.* metrics<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.filters
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetrics)</sup></sup>



Filters to apply towards all metrics collected by the collector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>allowList</b></td>
        <td>[]string</td>
        <td>
          List of metric patterns to allow<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>denyList</b></td>
        <td>[]string</td>
        <td>
          List of metric patterns to deny<br/>
          <br/>
            <i>Default</i>: [kubernetes.sys_container.* kubernetes.collector.runtime.* kubernetes.*.network.rx_rate kubernetes.*.network.rx_errors_rate kubernetes.*.network.tx_rate kubernetes.*.network.tx_errors_rate kubernetes.*.memory.page_faults kubernetes.*.memory.page_faults_rate kubernetes.*.memory.major_page_faults kubernetes.*.memory.major_page_faults_rate kubernetes.*.filesystem.inodes kubernetes.*.filesystem.inodes_free kubernetes.*.ephemeral_storage.request kubernetes.*.ephemeral_storage.limit]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tagGuaranteeList</b></td>
        <td>[]string</td>
        <td>
          List of tags guaranteed to not be removed during kubernetes metrics collection. Supersedes all other collection filters. These tags are given priority if you hit the 20 tag limit.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.nodeCollector
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetrics)</sup></sup>



NodeCollector is for resource configuration for the node collector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsnodecollectorresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the Collector containers.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.nodeCollector.resources
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsnodecollector)</sup></sup>



Resources Compute resources required by the Collector containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsnodecollectorresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdatacollectionmetricsnodecollectorresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.nodeCollector.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsnodecollectorresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.metrics.nodeCollector.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecdatacollectionmetricsnodecollectorresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataCollection.tolerations[index]
<sup><sup>[↩ Parent](#wavefrontspecdatacollection)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>enum</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
          <br/>
            <i>Enum</i>: NoSchedule, PreferNoSchedule, NoExecute<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>enum</td>
        <td>
          Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.<br/>
          <br/>
            <i>Enum</i>: Equal, Exists<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport
<sup><sup>[↩ Parent](#wavefrontspec)</sup></sup>



DataExport options

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdataexportexternalwavefrontproxy">externalWavefrontProxy</a></b></td>
        <td>object</td>
        <td>
          External Wavefront WavefrontProxy configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxy">wavefrontProxy</a></b></td>
        <td>object</td>
        <td>
          WavefrontProxy configuration options<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.externalWavefrontProxy
<sup><sup>[↩ Parent](#wavefrontspecdataexport)</sup></sup>



External Wavefront WavefrontProxy configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          Url is the proxy URL that the collector sends metrics to.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy
<sup><sup>[↩ Parent](#wavefrontspecdataexport)</sup></sup>



WavefrontProxy configuration options

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>args</b></td>
        <td>string</td>
        <td>
          Args is additional Wavefront proxy properties to be passed as command line arguments in the --<property_name> <value> format. Multiple properties can be specified.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deltaCounterPort</b></td>
        <td>integer</td>
        <td>
          DeltaCounterPort accumulates 1-minute delta counters on Wavefront data format (usually 50000)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable is whether to enable the wavefront proxy. Defaults to true.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyhistogram">histogram</a></b></td>
        <td>object</td>
        <td>
          Histogram distribution configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyhttpproxy">httpProxy</a></b></td>
        <td>object</td>
        <td>
          HttpProxy configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metricPort</b></td>
        <td>integer</td>
        <td>
          MetricPort is the primary port for Wavefront data format metrics. Defaults to 2878.<br/>
          <br/>
            <i>Default</i>: 2878<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyotlp">otlp</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetry Protocol configuration<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>preprocessor</b></td>
        <td>string</td>
        <td>
          Preprocessor is the name of the configmap containing a rules.yaml key with proxy preprocessing rules<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          Replicas number of replicas<br/>
          <br/>
            <i>Default</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the Proxy containers.<br/>
          <br/>
            <i>Default</i>: map[limits:map[cpu:1000m ephemeral-storage:8Gi memory:4Gi] requests:map[cpu:100m ephemeral-storage:2Gi memory:1Gi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxytracing">tracing</a></b></td>
        <td>object</td>
        <td>
          Distributed tracing configuration<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.histogram
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxy)</sup></sup>



Histogram distribution configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dayPort</b></td>
        <td>integer</td>
        <td>
          DayPort to accumulate 1-day based histograms on Wavefront data format (usually 40003)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hourPort</b></td>
        <td>integer</td>
        <td>
          HourPort to accumulate 1-hour based histograms on Wavefront data format (usually 40002)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minutePort</b></td>
        <td>integer</td>
        <td>
          MinutePort to accumulate 1-minute based histograms on Wavefront data format (usually 40001)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port for histogram distribution format data (usually 40000)<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.httpProxy
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxy)</sup></sup>



HttpProxy configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Name of the secret containing the HttpProxy configuration.<br/>
          <br/>
            <i>Default</i>: http-proxy-secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.otlp
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxy)</sup></sup>



OpenTelemetry Protocol configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>grpcPort</b></td>
        <td>integer</td>
        <td>
          GrpcPort for OTLP GRPC format data (usually 4317)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>httpPort</b></td>
        <td>integer</td>
        <td>
          HttpPort for OTLP format data (usually 4318)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>resourceAttrsOnMetricsIncluded</b></td>
        <td>boolean</td>
        <td>
          Enable resource attributes on metrics to be included. Defaults to false.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.resources
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxy)</sup></sup>



Resources Compute resources required by the Proxy containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxyresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxyresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxyresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.tracing
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxy)</sup></sup>



Distributed tracing configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxytracingjaeger">jaeger</a></b></td>
        <td>object</td>
        <td>
          Jaeger distributed tracing configurations<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxytracingwavefront">wavefront</a></b></td>
        <td>object</td>
        <td>
          Wavefront distributed tracing configurations<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecdataexportwavefrontproxytracingzipkin">zipkin</a></b></td>
        <td>object</td>
        <td>
          Zipkin distributed tracing configurations<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.tracing.jaeger
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxytracing)</sup></sup>



Jaeger distributed tracing configurations

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>applicationName</b></td>
        <td>string</td>
        <td>
          ApplicationName Custom application name for traces received on Jaeger's Http or Gprc port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>grpcPort</b></td>
        <td>integer</td>
        <td>
          GrpcPort for Jaeger GRPC format data (usually 14250)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>httpPort</b></td>
        <td>integer</td>
        <td>
          HttpPort for Jaeger Thrift format data (usually 30080)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port for Jaeger format distributed tracing data (usually 30001)<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.tracing.wavefront
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxytracing)</sup></sup>



Wavefront distributed tracing configurations

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port for distributed tracing data (usually 30000)<br/>
          <br/>
            <i>Default</i>: 30000<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>samplingDuration</b></td>
        <td>integer</td>
        <td>
          SamplingDuration When set to greater than 0, spans that exceed this duration will force trace to be sampled (ms)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>samplingRate</b></td>
        <td>string</td>
        <td>
          SamplingRate Distributed tracing data sampling rate (0 to 1)<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.dataExport.wavefrontProxy.tracing.zipkin
<sup><sup>[↩ Parent](#wavefrontspecdataexportwavefrontproxytracing)</sup></sup>



Zipkin distributed tracing configurations

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>applicationName</b></td>
        <td>string</td>
        <td>
          ApplicationName Custom application name for traces received on Zipkin's port.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port for Zipkin format distributed tracing data (usually 9411)<br/>
          <br/>
            <i>Default</i>: 9411<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental
<sup><sup>[↩ Parent](#wavefrontspec)</sup></sup>



Experimental features

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecexperimentalautotracing">autotracing</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalhub">hub</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalinsights">insights</a></b></td>
        <td>object</td>
        <td>
          Insights<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.autotracing
<sup><sup>[↩ Parent](#wavefrontspecexperimental)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalautotracingpem">pem</a></b></td>
        <td>object</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: map[resources:map[limits:map[cpu:1000m memory:600Mi] requests:map[cpu:100m memory:600Mi]]]<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.autotracing.pem
<sup><sup>[↩ Parent](#wavefrontspecexperimentalautotracing)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecexperimentalautotracingpemresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the Pem containers.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.autotracing.pem.resources
<sup><sup>[↩ Parent](#wavefrontspecexperimentalautotracingpem)</sup></sup>



Resources Compute resources required by the Pem containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecexperimentalautotracingpemresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalautotracingpemresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.autotracing.pem.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecexperimentalautotracingpemresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.autotracing.pem.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecexperimentalautotracingpemresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub
<sup><sup>[↩ Parent](#wavefrontspecexperimental)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalhubpixie">pixie</a></b></td>
        <td>object</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: map[enable:true]<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub.pixie
<sup><sup>[↩ Parent](#wavefrontspecexperimentalhub)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalhubpixiepem">pem</a></b></td>
        <td>object</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: map[resources:map[limits:map[cpu:1000m memory:2Gi] requests:map[cpu:100m memory:1Gi]]]<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub.pixie.pem
<sup><sup>[↩ Parent](#wavefrontspecexperimentalhubpixie)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecexperimentalhubpixiepemresources">resources</a></b></td>
        <td>object</td>
        <td>
          Resources Compute resources required by the Pem containers.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub.pixie.pem.resources
<sup><sup>[↩ Parent](#wavefrontspecexperimentalhubpixiepem)</sup></sup>



Resources Compute resources required by the Pem containers.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#wavefrontspecexperimentalhubpixiepemresourceslimits">limits</a></b></td>
        <td>object</td>
        <td>
          Limits CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontspecexperimentalhubpixiepemresourcesrequests">requests</a></b></td>
        <td>object</td>
        <td>
          Requests CPU and Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub.pixie.pem.resources.limits
<sup><sup>[↩ Parent](#wavefrontspecexperimentalhubpixiepemresources)</sup></sup>



Limits CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.hub.pixie.pem.resources.requests
<sup><sup>[↩ Parent](#wavefrontspecexperimentalhubpixiepemresources)</sup></sup>



Requests CPU and Memory requirements

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cpu</b></td>
        <td>string</td>
        <td>
          CPU is for specifying CPU requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ephemeral-storage</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>memory</b></td>
        <td>string</td>
        <td>
          Memory is for specifying Memory requirements<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.spec.experimental.insights
<sup><sup>[↩ Parent](#wavefrontspecexperimental)</sup></sup>



Insights

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ingestionUrl</b></td>
        <td>string</td>
        <td>
          Ingestion Url is the endpoint to send kubernetes events.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable is whether to enable Insights. Defaults to false.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.status
<sup><sup>[↩ Parent](#wavefront)</sup></sup>



WavefrontStatus defines the observed state of Wavefront

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Message is a human-readable message indicating details about all the deployment statuses.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#wavefrontstatusresourcestatusesindex">resourceStatuses</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          Status is a quick view of all the deployment statuses.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Wavefront.status.resourceStatuses[index]
<sup><sup>[↩ Parent](#wavefrontstatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Human readable message indicating details of the component.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          Computed running status. (available / desired )<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
