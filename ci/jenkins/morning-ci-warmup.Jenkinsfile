pipeline {
  agent any

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GCP_CREDS = credentials("GCP_CREDS")
  }

  triggers {
    // 8:00AM in Denver, M-F.
    cron '''TZ=America/Denver
00 08 * * 1-5'''
  }

  stages {
    stage('Prepare All Workers and Clusters') {
      parallel{
        stage("Lease worker-1") {
          agent {
            label "worker-1" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'echo Successfully created worker-1 with 1 day lease.'
          }
        }

        stage("Lease worker-2") {
          agent {
            label "worker-2" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'echo Successfully created worker-2 with 1 day lease.'
          }
        }

        stage("Lease worker-3") {
          agent {
            label "worker-3" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'echo Successfully created worker-3 with 1 day lease.'
          }
        }

        stage("Lease worker-4") {
          agent {
            label "worker-4" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'echo Successfully created worker-4 with 1 day lease.'
          }
        }

        stage("Lease worker-5") {
          agent {
            label "worker-5" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'echo Successfully created worker-5 with 1 day lease.'
          }
        }

        stage("Prepare gke-integration-worker and Cluster") {
          agent {
            label "gke-integration-worker" // TODO move test and publish to separate workers for parallel locks
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh 'make gke-connect-to-cluster'

            sh 'echo Successfully created gke-integration-worker with 1 day lease.'
          }
        }

        stage("Prepare gke-operator-worker-1 and Cluster") {
          agent {
            label "gke-operator-worker-1"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-1"
            GCP_ZONE="a"
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh './ci/jenkins/get-or-create-cluster.sh'

            sh 'echo Successfully created gke-operator-worker-1 with 1 day lease.'
          }
        }

        stage("Prepare gke-operator-worker-2 and Cluster") {
          agent {
            label "gke-operator-worker-2"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-2"
            GCP_ZONE="a"
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh './ci/jenkins/get-or-create-cluster.sh'

            sh 'echo Successfully created gke-operator-worker-2 with 1 day lease.'
          }
        }

        stage("Prepare gke-operator-worker-3 and Cluster") {
          agent {
            label "gke-operator-worker-3"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-3"
            GCP_ZONE="a"
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh './ci/jenkins/get-or-create-cluster.sh'

            sh 'echo Successfully created gke-operator-worker-3 with 1 day lease.'
          }
        }

        stage("Prepare eks-worker for Integration Tests") {
          agent {
            label "eks-integration-worker"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k eks'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
          }
        }

        stage("Prepare aks-worker for Integration Tests") {
          agent {
            label "aks-integration-worker"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
          }
          steps {
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k aks'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
          }
        }
      }
    }
  }
}
